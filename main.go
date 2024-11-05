package main

import (
	"bytes"
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

type AudioCacheEntry struct {
	data      []byte
	timestamp time.Time
}

var audioCache = sync.Map{}

func hashKey(text, lang string) string {
	h := fnv.New32a()
	h.Write([]byte(fmt.Sprintf("%s:%s", text, lang)))
	return fmt.Sprintf("%x", h.Sum32())
}

// Generates or retrieves cached audio data
func getOrGenerateAudio(text, lang string) ([]byte, error) {
	cacheKey := hashKey(text, lang)

	// Check in-memory cache first
	if entry, exists := audioCache.Load(cacheKey); exists {
		cachedEntry := entry.(AudioCacheEntry)
		return cachedEntry.data, nil
	}

	// Generate audio if not cached
	audioData, err := generateAudioData(text, lang)
	if err != nil {
		return nil, err
	}

	// Cache the generated audio
	audioCache.Store(cacheKey, AudioCacheEntry{data: audioData, timestamp: time.Now()})
	return audioData, nil
}

// Generate audio data without saving to disk
func generateAudioData(text, lang string) ([]byte, error) {
	cmd := exec.Command("gtts-cli", "--lang", lang, "--nocheck", text)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// CORS middleware to allow cross-origin requests
func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleSpeak(w http.ResponseWriter, r *http.Request) {
	var payload RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	audioData, err := getOrGenerateAudio(payload.Text, payload.Lang)
	if err != nil {
		http.Error(w, "Failed to generate audio", http.StatusInternalServerError)
		return
	}

	// Serve the audio content from memory
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(audioData)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/speak", handleSpeak)

	// Apply the CORS middleware
	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", enableCors(mux)))
}
