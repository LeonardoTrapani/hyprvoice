package transcriber

import (
	"context"
	"fmt"
	"os"

	"github.com/leonardotrapani/hyprvoice/internal/recording"
)

// Main transcriber interface
type Transcriber interface {
	Start(ctx context.Context, frameCh <-chan recording.AudioFrame) (<-chan error, error)
	Stop(ctx context.Context) error
	GetFinalTranscription() (string, error)
}

// Adapter interface for different transcription backends
type TranscriptionAdapter interface {
	Transcribe(ctx context.Context, audioData []byte) (string, error)
}

// Configuration for the transcriber
type Config struct {
	Provider string
	APIKey   string
	Language string
	Model    string
}

func DefaultConfig() Config {
	return Config{
		Provider: "openai",
		Language: "it",
		Model:    "whisper-1",
	}
}

// NewTranscriber creates a new simple transcriber
func NewTranscriber(config Config) (Transcriber, error) {
	// Create the appropriate adapter
	var adapter TranscriptionAdapter

	switch config.Provider {
	case "openai":
		if config.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key required")
		}
		adapter = NewOpenAIAdapter(config)

	case "whisper.cpp":
		adapter = NewWhisperCppAdapter(config)

	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	// Create simple transcriber that collects all audio
	transcriber := NewSimpleTranscriber(config, adapter)

	return transcriber, nil
}

func NewDefaultTranscriber() (Transcriber, error) {
	config := DefaultConfig()
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}

	return NewTranscriber(config)
}
