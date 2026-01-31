package provider

import (
	"strings"

	"github.com/leonardotrapani/hyprvoice/internal/language"
)

// GroqProvider implements Provider for Groq services
type GroqProvider struct{}

func (p *GroqProvider) Name() string {
	return "groq"
}

func (p *GroqProvider) RequiresAPIKey() bool {
	return true
}

func (p *GroqProvider) ValidateAPIKey(key string) bool {
	return strings.HasPrefix(key, "gsk_")
}

func (p *GroqProvider) IsLocal() bool {
	return false
}

func (p *GroqProvider) Models() []Model {
	allLangs := language.AllLanguageCodes()

	return []Model{
		// transcription models
		{
			ID:                 "whisper-large-v3",
			Name:               "Whisper Large v3",
			Description:        "Full Whisper v3 model, best accuracy",
			Type:               Transcription,
			Streaming:          false,
			Local:              false,
			AdapterType:        "openai",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.groq.com/openai", Path: "/v1/audio/transcriptions"},
		},
		{
			ID:                 "whisper-large-v3-turbo",
			Name:               "Whisper Large v3 Turbo",
			Description:        "Faster Whisper v3 with slightly lower accuracy",
			Type:               Transcription,
			Streaming:          false,
			Local:              false,
			AdapterType:        "openai",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.groq.com/openai", Path: "/v1/audio/transcriptions"},
		},
		{
			ID:                 "distil-whisper-large-v3-en",
			Name:               "Distil Whisper Large v3 EN",
			Description:        "English-only, fastest option",
			Type:               Transcription,
			Streaming:          false,
			Local:              false,
			AdapterType:        "openai",
			SupportedLanguages: []string{"en"},
			Endpoint:           &EndpointConfig{BaseURL: "https://api.groq.com/openai", Path: "/v1/audio/transcriptions"},
		},
		// LLM models
		{
			ID:                 "llama-3.3-70b-versatile",
			Name:               "Llama 3.3 70B Versatile",
			Description:        "Most capable Llama model",
			Type:               LLM,
			Streaming:          false,
			Local:              false,
			AdapterType:        "openai",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.groq.com/openai", Path: "/v1/chat/completions"},
		},
		{
			ID:                 "llama-3.1-8b-instant",
			Name:               "Llama 3.1 8B Instant",
			Description:        "Fast and efficient",
			Type:               LLM,
			Streaming:          false,
			Local:              false,
			AdapterType:        "openai",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.groq.com/openai", Path: "/v1/chat/completions"},
		},
		{
			ID:                 "mixtral-8x7b-32768",
			Name:               "Mixtral 8x7B",
			Description:        "Mixture of experts model",
			Type:               LLM,
			Streaming:          false,
			Local:              false,
			AdapterType:        "openai",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.groq.com/openai", Path: "/v1/chat/completions"},
		},
	}
}

func (p *GroqProvider) DefaultModel(t ModelType) string {
	switch t {
	case Transcription:
		return "whisper-large-v3-turbo"
	case LLM:
		return "llama-3.3-70b-versatile"
	}
	return ""
}
