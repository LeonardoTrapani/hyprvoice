package transcriber

import (
	"context"
	"fmt"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
	"github.com/leonardotrapani/hyprvoice/internal/recording"
)

// Main transcriber interface
type Transcriber interface {
	Start(ctx context.Context, frameCh <-chan recording.AudioFrame) (<-chan error, error)
	Stop(ctx context.Context) error
	GetFinalTranscription() (string, error)
}

// BatchAdapter interface for batch transcription backends (collect all audio, transcribe at end)
type BatchAdapter interface {
	Transcribe(ctx context.Context, audioData []byte) (string, error)
}

// Configuration for the transcriber
type Config struct {
	Provider string
	APIKey   string
	Language string
	Model    string
	Keywords []string
}

// NewTranscriber creates a new simple transcriber
func NewTranscriber(config Config) (Transcriber, error) {
	// Create the appropriate adapter
	var adapter BatchAdapter

	switch config.Provider {
	case "openai":
		if config.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key required")
		}
		adapter = NewOpenAIAdapterFromConfig(config)

	case "groq-transcription":
		if config.APIKey == "" {
			return nil, fmt.Errorf("Groq API key required")
		}
		// use consolidated OpenAI adapter with Groq endpoint
		endpoint := &provider.EndpointConfig{BaseURL: "https://api.groq.com/openai"}
		adapter = NewOpenAIAdapter(endpoint, config.APIKey, config.Model, config.Language, config.Keywords, "groq")

	case "groq-translation":
		if config.APIKey == "" {
			return nil, fmt.Errorf("Groq API key required")
		}
		adapter = NewGroqTranslationAdapter(config)

	case "mistral-transcription":
		if config.APIKey == "" {
			return nil, fmt.Errorf("Mistral API key required")
		}
		// use consolidated OpenAI adapter with Mistral endpoint
		endpoint := &provider.EndpointConfig{BaseURL: "https://api.mistral.ai"}
		adapter = NewOpenAIAdapter(endpoint, config.APIKey, config.Model, config.Language, config.Keywords, "mistral")

	case "elevenlabs":
		if config.APIKey == "" {
			return nil, fmt.Errorf("ElevenLabs API key required")
		}
		adapter = NewElevenLabsAdapterFromConfig(config)

	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	// Create simple transcriber that collects all audio
	transcriber := NewSimpleTranscriber(config, adapter)

	return transcriber, nil
}
