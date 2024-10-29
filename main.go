package main

import (
	"encoding/json"
	"log"
	"net/http"
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

func handleSpeak(w http.ResponseWriter, r *http.Request) {
	var payload RequestPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Call gTTS using the text and language
	err = generateAudioFile(payload.Text, payload.Lang)
	if err != nil {
		http.Error(w, "Failed to generate audio", http.StatusInternalServerError)
		return
	}

	// Serve the audio file to the client
	http.ServeFile(w, r, "output.mp3")
}

func main() {
	http.HandleFunc("/speak", handleSpeak)
	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
