package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
)

type RequestPayload struct {
	Text string `json:"text"`
	Lang string `json:"lang"`
}

var audioCache = sync.Map{}

func hashKey(text, lang string) string {
	h := fnv.New32a()
	h.Write([]byte(fmt.Sprintf("%s:%s", text, lang)))
	return fmt.Sprintf("%x", h.Sum32())
}

func generateAudioFile(text, lang string) (string, error) {
	cacheKey := hashKey(text, lang)
	filename := fmt.Sprintf("output_%s.mp3", cacheKey)

	if _, exists := audioCache.Load(cacheKey); exists {
		return filename, nil
	}

	cmd := exec.Command("gtts-cli", "--lang", lang, "--nocheck", "--output", filename, text)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("gTTS error: %s\n", string(output))
		return "", err
	}

	audioCache.Store(cacheKey, filename)
	return filename, nil
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

	filename, err := generateAudioFile(payload.Text, payload.Lang)
	if err != nil {
		http.Error(w, "Failed to generate audio", http.StatusInternalServerError)
		return
	}

	file, err := os.Open(filename)
	if err != nil {
		http.Error(w, "Failed to open audio file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to retrieve file information", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeContent(w, r, filename, fileInfo.ModTime(), file)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/speak", handleSpeak)

	// Apply the CORS middleware
	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", enableCors(mux)))
}
