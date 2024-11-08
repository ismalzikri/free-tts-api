package main

import (
	"bytes"
	"container/list"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

type RequestPayload struct {
	Text string `json:"text"`
	Lang string `json:"lang"`
}

type ResponsePayload struct {
	Audio string `json:"audio"` // Base64 encoded audio data
}

type AudioCacheEntry struct {
	data      []byte
	timestamp time.Time
}

// Cache manager with LRU and expiration
type AudioCache struct {
	cache      map[string]*list.Element
	expiration time.Duration
	maxSize    int
	mu         sync.Mutex
	lruList    *list.List
}

type cacheItem struct {
	key   string
	entry AudioCacheEntry
}

// NewAudioCache creates a cache with a specified max size and expiration time
func NewAudioCache(maxSize int, expiration time.Duration) *AudioCache {
	cache := &AudioCache{
		cache:      make(map[string]*list.Element),
		expiration: expiration,
		maxSize:    maxSize,
		lruList:    list.New(),
	}
	go cache.evictExpiredEntries()
	return cache
}

func (c *AudioCache) evictExpiredEntries() {
	for {
		time.Sleep(c.expiration)
		c.mu.Lock()
		for key, elem := range c.cache {
			if time.Since(elem.Value.(cacheItem).entry.timestamp) > c.expiration {
				c.remove(key)
			}
		}
		c.mu.Unlock()
	}
}

func (c *AudioCache) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, exists := c.cache[key]; exists {
		c.lruList.MoveToFront(elem)
		return elem.Value.(cacheItem).entry.data, true
	}
	return nil, false
}

func (c *AudioCache) set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, exists := c.cache[key]; exists {
		c.lruList.MoveToFront(elem)
		elem.Value = cacheItem{key: key, entry: AudioCacheEntry{data: data, timestamp: time.Now()}}
	} else {
		if c.lruList.Len() >= c.maxSize {
			oldest := c.lruList.Back()
			if oldest != nil {
				c.remove(oldest.Value.(cacheItem).key)
			}
		}
		entry := AudioCacheEntry{data: data, timestamp: time.Now()}
		elem := c.lruList.PushFront(cacheItem{key: key, entry: entry})
		c.cache[key] = elem
	}
}

func (c *AudioCache) remove(key string) {
	if elem, exists := c.cache[key]; exists {
		delete(c.cache, key)
		c.lruList.Remove(elem)
	}
}

// Helper function to hash the text and language
func hashKey(text, lang string) string {
	h := fnv.New32a()
	h.Write([]byte(fmt.Sprintf("%s:%s", text, lang)))
	return fmt.Sprintf("%x", h.Sum32())
}

// Generate or retrieve audio from cache
func getOrGenerateAudio(text, lang string, cache *AudioCache) ([]byte, error) {
	cacheKey := hashKey(text, lang)

	// Check in-memory cache first
	if data, exists := cache.get(cacheKey); exists {
		return data, nil
	}

	// Generate audio if not cached
	audioData, err := generateAudioData(text, lang)
	if err != nil {
		return nil, err
	}

	// Cache the generated audio
	cache.set(cacheKey, audioData)
	return audioData, nil
}

// Generate audio data without saving to disk
func generateAudioData(text, lang string) ([]byte, error) {
	// Generate audio using gTTS CLI
	gttsCmd := exec.Command("gtts-cli", "--lang", lang, "--nocheck", text)
	var gttsOut bytes.Buffer
	gttsCmd.Stdout = &gttsOut
	if err := gttsCmd.Run(); err != nil {
		return nil, err
	}

	// Pipe gTTS output to ffmpeg for Opus encoding
	ffmpegCmd := exec.Command("ffmpeg", "-f", "wav", "-i", "pipe:0", "-c:a", "libopus", "-b:a", "32k", "-f", "opus", "pipe:1")
	ffmpegCmd.Stdin = &gttsOut
	var opusOut bytes.Buffer
	ffmpegCmd.Stdout = &opusOut
	if err := ffmpegCmd.Run(); err != nil {
		return nil, err
	}

	return opusOut.Bytes(), nil
}

// CORS middleware to allow cross-origin requests
func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleSpeak(w http.ResponseWriter, r *http.Request, cache *AudioCache) {
	var payload RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	audioData, err := getOrGenerateAudio(payload.Text, payload.Lang, cache)
	if err != nil {
		http.Error(w, "Failed to generate audio", http.StatusInternalServerError)
		return
	}

	// Convert audio data to Base64 string
	base64Audio := base64.StdEncoding.EncodeToString(audioData)

	// Send the Base64-encoded audio in JSON format
	responsePayload := ResponsePayload{Audio: base64Audio}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responsePayload)
}

func main() {
	audioCache := NewAudioCache(100, 3*time.Hour) // Max 100 items, 3-hour expiration
	mux := http.NewServeMux()
	mux.HandleFunc("/speak", func(w http.ResponseWriter, r *http.Request) {
		handleSpeak(w, r, audioCache)
	})

	// Apply the CORS middleware
	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", enableCors(mux)))
}
