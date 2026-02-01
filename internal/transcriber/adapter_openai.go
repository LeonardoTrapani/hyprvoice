package transcriber

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
	"github.com/sashabaranov/go-openai"
)

// OpenAIAdapter implements BatchAdapter for any OpenAI-compatible API
// Works with OpenAI, Groq, Mistral, and any other OpenAI-compatible endpoint
type OpenAIAdapter struct {
	client       *openai.Client
	model        string
	language     string
	keywords     []string
	providerName string
}

// NewOpenAIAdapter creates an adapter for OpenAI-compatible transcription APIs
// endpoint: the BaseURL for the API (e.g., "https://api.openai.com", "https://api.groq.com/openai")
// apiKey: the API key for authentication
// model: model ID to use
// lang: provider language code
// keywords: optional spelling hints
// providerName: used for logging and language format conversion
func NewOpenAIAdapter(endpoint *provider.EndpointConfig, apiKey, model, lang string, keywords []string, providerName string) *OpenAIAdapter {
	var client *openai.Client

	if endpoint != nil && endpoint.BaseURL != "" {
		// use custom endpoint
		clientConfig := openai.DefaultConfig(apiKey)
		clientConfig.BaseURL = endpoint.BaseURL + "/v1"
		client = openai.NewClientWithConfig(clientConfig)
	} else {
		// default to OpenAI
		client = openai.NewClient(apiKey)
	}

	return &OpenAIAdapter{
		client:       client,
		model:        model,
		language:     lang,
		keywords:     keywords,
		providerName: providerName,
	}
}

// NewOpenAIAdapterFromConfig creates an adapter using the legacy Config struct
// This is for backwards compatibility during migration
func NewOpenAIAdapterFromConfig(config Config) *OpenAIAdapter {
	return NewOpenAIAdapter(nil, config.APIKey, config.Model, config.Language, config.Keywords, "openai")
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
		Model:    a.model,
		Reader:   bytes.NewReader(wavData),
		FilePath: "audio.wav",
		Language: a.language,
	}

	// Add keywords as initial_prompt to help with spelling hints
	if len(a.keywords) > 0 {
		req.Prompt = strings.Join(a.keywords, ", ")
	}

	start := time.Now()
	resp, err := a.client.CreateTranscription(ctx, req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("%s-adapter: API call failed after %v: %v", a.providerName, duration, err)
		return "", fmt.Errorf("%s transcription: %w", a.providerName, err)
	}

	log.Printf("%s-adapter: transcribed %d bytes in %v: %q", a.providerName, len(audioData), duration, resp.Text)
	return resp.Text, nil
}
