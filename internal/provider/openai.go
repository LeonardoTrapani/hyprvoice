package provider

import (
	"strings"

	"github.com/leonardotrapani/hyprvoice/internal/language"
)

// OpenAIProvider implements Provider for OpenAI services
type OpenAIProvider struct{}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) RequiresAPIKey() bool {
	return true
}

func (p *OpenAIProvider) ValidateAPIKey(key string) bool {
	return strings.HasPrefix(key, "sk-")
}

func (p *OpenAIProvider) IsLocal() bool {
	return false
}

func (p *OpenAIProvider) Models() []Model {
	allLangs := language.AllLanguageCodes()

	return []Model{
		// transcription models
		{
			ID:                 "whisper-1",
			Name:               "Whisper 1",
			Description:        "OpenAI's production speech-to-text model",
			Type:               Transcription,
			Streaming:          false,
			Local:              false,
			AdapterType:        "openai",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.openai.com", Path: "/v1/audio/transcriptions"},
		},
		// LLM models
		{
			ID:                 "gpt-4o-mini",
			Name:               "GPT-4o Mini",
			Description:        "Fast and affordable GPT-4 variant",
			Type:               LLM,
			Streaming:          false,
			Local:              false,
			AdapterType:        "openai",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.openai.com", Path: "/v1/chat/completions"},
		},
		{
			ID:                 "gpt-4o",
			Name:               "GPT-4o",
			Description:        "Most capable GPT-4 model",
			Type:               LLM,
			Streaming:          false,
			Local:              false,
			AdapterType:        "openai",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.openai.com", Path: "/v1/chat/completions"},
		},
		{
			ID:                 "gpt-4-turbo",
			Name:               "GPT-4 Turbo",
			Description:        "Faster GPT-4 with large context window",
			Type:               LLM,
			Streaming:          false,
			Local:              false,
			AdapterType:        "openai",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.openai.com", Path: "/v1/chat/completions"},
		},
		{
			ID:                 "gpt-3.5-turbo",
			Name:               "GPT-3.5 Turbo",
			Description:        "Fast and cost-effective",
			Type:               LLM,
			Streaming:          false,
			Local:              false,
			AdapterType:        "openai",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.openai.com", Path: "/v1/chat/completions"},
		},
	}
}

func (p *OpenAIProvider) DefaultModel(t ModelType) string {
	switch t {
	case Transcription:
		return "whisper-1"
	case LLM:
		return "gpt-4o-mini"
	}
	return ""
}
