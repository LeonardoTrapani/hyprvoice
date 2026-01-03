package transcriber

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
)

// MistralAdapter implements TranscriptionAdapter for Mistral Voxtral API
type MistralAdapter struct {
	client *openai.Client
	config Config
}

func NewMistralAdapter(config Config) *MistralAdapter {
	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = "https://api.mistral.ai/v1"
	client := openai.NewClientWithConfig(clientConfig)

	return &MistralAdapter{
		client: client,
		config: config,
	}
}

func (a *MistralAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
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

	start := time.Now()
	resp, err := a.client.CreateTranscription(ctx, req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("mistral-adapter: API call failed after %v: %v", duration, err)
		return "", fmt.Errorf("mistral transcription: %w", err)
	}

	log.Printf("mistral-adapter: transcribed %d bytes in %v: %q", len(audioData), duration, resp.Text)
	return resp.Text, nil
}
