package transcriber

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
)

// GroqTranslationAdapter implements TranscriptionAdapter for Groq Translation API
// Translates audio to English text. The Language field in config hints at the source language.
type GroqTranslationAdapter struct {
	client *openai.Client
	config Config
}

func NewGroqTranslationAdapter(config Config) *GroqTranslationAdapter {
	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = "https://api.groq.com/openai/v1"
	client := openai.NewClientWithConfig(clientConfig)

	return &GroqTranslationAdapter{
		client: client,
		config: config,
	}
}

func (a *GroqTranslationAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if len(audioData) == 0 {
		return "", nil
	}

	// Convert raw PCM to WAV format
	wavData, err := convertToWAV(audioData)
	if err != nil {
		return "", fmt.Errorf("convert to WAV: %w", err)
	}

	// Create translation request
	// Note: Translation always outputs English, regardless of target language
	// The Language field in the request hints at the source audio language for better accuracy
	req := openai.AudioRequest{
		Model:    a.config.Model,
		Reader:   bytes.NewReader(wavData),
		FilePath: "audio.wav",
		Language: a.config.Language, // Source language hint
	}

	start := time.Now()
	resp, err := a.client.CreateTranslation(ctx, req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("groq-translation-adapter: API call failed after %v: %v", duration, err)
		return "", fmt.Errorf("groq translation: %w", err)
	}

	log.Printf("groq-translation-adapter: translated %d bytes in %v: %q", len(audioData), duration, resp.Text)
	return resp.Text, nil
}
