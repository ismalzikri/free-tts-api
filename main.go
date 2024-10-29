package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
)

type RequestPayload struct {
	Text string `json:"text"`
	Lang string `json:"lang"`
}

func generateAudioFile(text, lang string) error {
	// Construct the gTTS command
	cmd := exec.Command("gtts-cli", "--lang", lang, "--output", "output.mp3", text)

	// Capture output for error handling
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("gTTS error: %s\n", string(output))
		return err
	}
	return nil
}

// CORS middleware to allow cross-origin requests
func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://open-reasearch-color-detection.vercel.app")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleSpeak(w http.ResponseWriter, r *http.Request) {
	var payload RequestPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Generate the audio file
	err = generateAudioFile(payload.Text, payload.Lang)
	if err != nil {
		http.Error(w, "Failed to generate audio", http.StatusInternalServerError)
		return
	}

	// Verify the file exists
	if _, err := os.Stat("output.mp3"); os.IsNotExist(err) {
		http.Error(w, "Audio file not found", http.StatusInternalServerError)
		return
	}

	// Set the Content-Type to audio/mpeg for MP3
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Disposition", "attachment; filename=output.mp3")

	// Serve the audio file to the client
	http.ServeFile(w, r, "output.mp3")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/speak", handleSpeak)

	// Apply the CORS middleware
	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", enableCors(mux)))
}
