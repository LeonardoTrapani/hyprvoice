package provider

import "strings"

// GroqProvider implements Provider for Groq services
type GroqProvider struct{}

func (p *GroqProvider) Name() string {
	return ProviderGroq
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
	// https://console.groq.com/docs/speech-to-text#supported-languages
	allLangs := groqTranscriptionLanguages
	docsURL := "https://console.groq.com/docs/speech-to-text#supported-languages"

	return []Model{
		// transcription models
		{
			ID:                 "whisper-large-v3",
			Name:               "Whisper Large v3",
			Description:        "Full Whisper v3 model, best accuracy",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              false,
			AdapterType:        AdapterOpenAI,
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.groq.com/openai", Path: "/v1/audio/transcriptions"},
			DocsURL:            docsURL,
		},
		{
			ID:                 "whisper-large-v3-turbo",
			Name:               "Whisper Large v3 Turbo",
			Description:        "Faster Whisper v3 with slightly lower accuracy",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              false,
			AdapterType:        AdapterOpenAI,
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.groq.com/openai", Path: "/v1/audio/transcriptions"},
			DocsURL:            docsURL,
		},
		// LLM models
		{
			ID:                "llama-3.3-70b-versatile",
			Name:              "Llama 3.3 70B Versatile",
			Description:       "Most capable Llama model",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenAI,
			Endpoint:          &EndpointConfig{BaseURL: "https://api.groq.com/openai", Path: "/v1/chat/completions"},
		},
		{
			ID:                "llama-3.1-8b-instant",
			Name:              "Llama 3.1 8B Instant",
			Description:       "Fast and efficient",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenAI,
			Endpoint:          &EndpointConfig{BaseURL: "https://api.groq.com/openai", Path: "/v1/chat/completions"},
		},
		{
			ID:                "mixtral-8x7b-32768",
			Name:              "Mixtral 8x7B",
			Description:       "Mixture of experts model",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenAI,
			Endpoint:          &EndpointConfig{BaseURL: "https://api.groq.com/openai", Path: "/v1/chat/completions"},
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
