package transcriber

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

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
	Provider  string
	APIKey    string
	Language  string
	Model     string
	Keywords  []string
	Threads   int  // CPU threads for local transcription (0 = auto)
	Streaming bool // use streaming mode if model supports it
}

// NewTranscriber creates a new transcriber based on model metadata
func NewTranscriber(config Config) (Transcriber, error) {
	if config.Provider == "" {
		return nil, fmt.Errorf("provider is required")
	}

	// map config provider name to registry provider name
	registryProvider := provider.BaseProviderName(config.Provider)

	// lookup provider
	p := provider.GetProvider(registryProvider)
	if p == nil {
		return nil, fmt.Errorf("unknown provider: %s", config.Provider)
	}

	// check API key requirement
	if p.RequiresAPIKey() && config.APIKey == "" {
		return nil, fmt.Errorf("%s API key required", cases.Title(language.English).String(registryProvider))
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

	// runtime language-model compatibility check with fallback
	// primary validation happens at config time (hard error), this is a safety net
	if config.Language != "" && !model.SupportsLanguage(config.Language) {
		log.Printf("warning: model %s does not support language %s, falling back to auto-detect", model.ID, config.Language)
		config.Language = ""
	}

	// determine if we should use streaming mode
	useStreaming := config.Streaming && model.SupportsStreaming

	// fail if streaming-only model is used without streaming enabled
	if !useStreaming && !model.SupportsBatch {
		return nil, fmt.Errorf("model %s requires streaming mode (set streaming = true in config)", model.ID)
	}

	// streaming mode: use StreamingTranscriber
	if useStreaming {
		// pick the right adapter type for streaming
		adapterType := model.AdapterType
		if model.StreamingAdapter != "" {
			adapterType = model.StreamingAdapter
		}

		// pick the right endpoint for streaming
		endpoint := model.Endpoint
		if model.StreamingEndpoint != nil {
			endpoint = model.StreamingEndpoint
		}

		var streamingAdapter StreamingAdapter
		switch adapterType {
		case provider.AdapterElevenLabsStream:
			streamingAdapter = NewElevenLabsStreamingAdapter(endpoint, config.APIKey, model.ID, config.Language, config.Keywords)
		case provider.AdapterDeepgram:
			streamingAdapter = NewDeepgramAdapter(endpoint, config.APIKey, model.ID, config.Language, config.Keywords)
		case provider.AdapterOpenAIRealtime:
			streamingAdapter = NewOpenAIRealtimeAdapter(endpoint, config.APIKey, model.ID, config.Language, config.Keywords)
		default:
			return nil, fmt.Errorf("unsupported streaming adapter type: %s", adapterType)
		}
		return NewStreamingTranscriber(streamingAdapter, config.Language), nil
	}

	// batch mode: use SimpleTranscriber
	var adapter BatchAdapter
	switch model.AdapterType {
	case provider.AdapterOpenAI:
		adapter = NewOpenAIAdapter(model.Endpoint, config.APIKey, model.ID, config.Language, config.Keywords, registryProvider)
	case provider.AdapterElevenLabs:
		adapter = NewElevenLabsAdapter(model.Endpoint, config.APIKey, model.ID, config.Language, config.Keywords)
	case provider.AdapterDeepgram:
		adapter = NewDeepgramBatchAdapter(model.Endpoint, config.APIKey, model.ID, config.Language, config.Keywords)
	case provider.AdapterWhisperCpp:
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
