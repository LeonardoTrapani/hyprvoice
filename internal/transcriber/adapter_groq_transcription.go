package transcriber

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// GroqTranscriptionAdapter implements BatchAdapter for Groq Whisper API
type GroqTranscriptionAdapter struct {
	client *openai.Client
	config Config
}

func NewGroqTranscriptionAdapter(config Config) *GroqTranscriptionAdapter {
	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = "https://api.groq.com/openai/v1"
	client := openai.NewClientWithConfig(clientConfig)

	return &GroqTranscriptionAdapter{
		client: client,
		config: config,
	}
}

func (a *GroqTranscriptionAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if len(audioData) == 0 {
		return "", nil
	}

	// Convert raw PCM to WAV format
	wavData, err := convertToWAV(audioData)
	if err != nil {
		return "", fmt.Errorf("convert to WAV: %w", err)
	}

	// Create transcription request
	req := openai.AudioRequest{
		Model:    a.config.Model,
		Reader:   bytes.NewReader(wavData),
		FilePath: "audio.wav",
		Language: a.config.Language,
	}

	// Add keywords as prompt to help with spelling hints
	if len(a.config.Keywords) > 0 {
		req.Prompt = strings.Join(a.config.Keywords, ", ")
	}

	start := time.Now()
	resp, err := a.client.CreateTranscription(ctx, req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("groq-transcription-adapter: API call failed after %v: %v", duration, err)
		return "", fmt.Errorf("groq transcription: %w", err)
	}

	log.Printf("groq-transcription-adapter: transcribed %d bytes in %v: %q", len(audioData), duration, resp.Text)
	return resp.Text, nil
}
