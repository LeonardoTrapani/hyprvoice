package transcriber

import (
	"context"
	"fmt"
	"strings"

	"github.com/leonardotrapani/hyprvoice/internal/models/whisper"
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
	Threads  int // CPU threads for local transcription (0 = auto)
}

// mapConfigProviderToRegistryName maps config provider names to provider registry names
// Config uses names like "groq-transcription", "groq-translation", "mistral-transcription"
// Registry uses base names like "groq", "mistral"
func mapConfigProviderToRegistryName(configProvider string) string {
	switch configProvider {
	case "groq-transcription", "groq-translation":
		return "groq"
	case "mistral-transcription":
		return "mistral"
	default:
		return configProvider
	}
}

// NewTranscriber creates a new transcriber based on model metadata
func NewTranscriber(config Config) (Transcriber, error) {
	if config.Provider == "" {
		return nil, fmt.Errorf("provider is required")
	}

	// special case: groq-translation uses CreateTranslation API (different from transcription)
	if config.Provider == "groq-translation" {
		if config.APIKey == "" {
			return nil, fmt.Errorf("Groq API key required")
		}
		adapter := NewGroqTranslationAdapter(config)
		return NewSimpleTranscriber(config, adapter), nil
	}

	// map config provider name to registry provider name
	registryProvider := mapConfigProviderToRegistryName(config.Provider)

	// lookup provider
	p := provider.GetProvider(registryProvider)
	if p == nil {
		return nil, fmt.Errorf("unknown provider: %s", config.Provider)
	}

	// check API key requirement
	if p.RequiresAPIKey() && config.APIKey == "" {
		return nil, fmt.Errorf("%s API key required", strings.Title(registryProvider))
	}

	// lookup model from provider
	model, err := provider.GetModel(registryProvider, config.Model)
	if err != nil {
		// if model not found, try to use default model
		if config.Model == "" {
			defaultModel := p.DefaultModel(provider.Transcription)
			if defaultModel != "" {
				model, err = provider.GetModel(registryProvider, defaultModel)
			}
		}
		if err != nil || model == nil {
			return nil, fmt.Errorf("model not found: %s (provider: %s)", config.Model, config.Provider)
		}
	}

	// check model type
	if model.Type != provider.Transcription {
		return nil, fmt.Errorf("model %s is not a transcription model", config.Model)
	}

	// streaming models not supported yet
	if model.Streaming {
		return nil, fmt.Errorf("streaming model %s not supported yet (coming soon)", config.Model)
	}

	// create adapter based on model.AdapterType
	var adapter BatchAdapter
	switch model.AdapterType {
	case "openai":
		adapter = NewOpenAIAdapter(model.Endpoint, config.APIKey, model.ID, config.Language, config.Keywords, registryProvider)
	case "elevenlabs":
		adapter = NewElevenLabsAdapter(model.Endpoint, config.APIKey, model.ID, config.Language)
	case "whisper-cpp":
		modelPath := whisper.GetModelPath(config.Model)
		if modelPath == "" {
			return nil, fmt.Errorf("unknown whisper model: %s", config.Model)
		}
		adapter = NewWhisperCppAdapter(modelPath, config.Language, config.Threads)
	default:
		return nil, fmt.Errorf("unsupported adapter type: %s", model.AdapterType)
	}

	return NewSimpleTranscriber(config, adapter), nil
}
