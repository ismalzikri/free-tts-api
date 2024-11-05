package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"os/exec"
	"sync"
)

type RequestPayload struct {
	Text string `json:"text"`
	Lang string `json:"lang"`
}

// Cache to store generated audio files
var audioCache = sync.Map{}

func hashKey(text, lang string) string {
	h := fnv.New32a()
	h.Write([]byte(fmt.Sprintf("%s:%s", text, lang)))
	return fmt.Sprintf("%x", h.Sum32())
}

// Generates or retrieves cached audio file
func generateAudioFile(text, lang string) (string, error) {
	cacheKey := hashKey(text, lang)
	filename := fmt.Sprintf("output_%s.mp3", cacheKey)

	// Check cache
	if _, exists := audioCache.Load(cacheKey); exists {
		return filename, nil
	}

	// Execute gTTS command to generate audio
	cmd := exec.Command("gtts-cli", "--lang", lang, "--nocheck", "--output", filename, text)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("gTTS error: %s\n", string(output))
		return "", err
	}

	// Store the file in cache and return filename
	audioCache.Store(cacheKey, filename)
	return filename, nil
}

// CORS middleware to allow cross-origin requests
func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://open-reasearch-color-detection.vercel.app")
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

	cacheKey := hashKey(payload.Text, payload.Lang)
	filename, err := generateAudioFile(payload.Text, payload.Lang)
	if err != nil {
		http.Error(w, "Failed to generate audio", http.StatusInternalServerError)
		return
	}

	// Handle conditional request using ETag
	eTag := fmt.Sprintf(`"%s"`, cacheKey)
	if match := r.Header.Get("If-None-Match"); match == eTag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Set response headers for caching
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Disposition", "attachment; filename=output.mp3")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	w.Header().Set("ETag", eTag)

	// Serve the audio file to the client
	http.ServeFile(w, r, filename)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/speak", handleSpeak)

	// Apply the CORS middleware
	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", enableCors(mux)))
}
