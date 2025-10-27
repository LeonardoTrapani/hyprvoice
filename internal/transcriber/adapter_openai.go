package transcriber

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
)

// OpenAIAdapter implements TranscriptionAdapter for OpenAI Whisper API
type OpenAIAdapter struct {
	client *openai.Client
	config Config
}

func NewOpenAIAdapter(config Config) *OpenAIAdapter {
	client := openai.NewClient(config.APIKey)
	return &OpenAIAdapter{
		client: client,
		config: config,
	}
}

func (a *OpenAIAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
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
		log.Printf("openai-adapter: API call failed after %v: %v", duration, err)
		return "", fmt.Errorf("openai transcription: %w", err)
	}

	log.Printf("openai-adapter: transcribed %d bytes in %v: %q", len(audioData), duration, resp.Text)
	return resp.Text, nil
}
