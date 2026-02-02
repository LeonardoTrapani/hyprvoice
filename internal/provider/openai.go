package provider

import "strings"

// OpenAIProvider implements Provider for OpenAI services
type OpenAIProvider struct{}

func (p *OpenAIProvider) Name() string {
	return ProviderOpenAI
}

func (p *OpenAIProvider) RequiresAPIKey() bool {
	return true
}

func (p *OpenAIProvider) ValidateAPIKey(key string) bool {
	return strings.HasPrefix(key, "sk-")
}

func (p *OpenAIProvider) APIKeyURL() string {
	return "https://platform.openai.com/api-keys"
}

func (p *OpenAIProvider) IsLocal() bool {
	return false
}

func (p *OpenAIProvider) Models() []Model {
	// https://platform.openai.com/docs/guides/speech-to-text#supported-languages
	allLangs := openaiTranscriptionLanguages

	docsURL := "https://platform.openai.com/docs/guides/speech-to-text#supported-languages"

	return []Model{
		// transcription models
		{
			ID:                 "whisper-1",
			Name:               "Whisper 1",
			Description:        "Reliable and cost-effective; good default for most use cases",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              false,
			AdapterType:        AdapterOpenAI,
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.openai.com", Path: "/v1/audio/transcriptions"},
			DocsURL:            docsURL,
		},
		{
			ID:                 "gpt-4o-transcribe",
			Name:               "GPT-4o Transcribe",
			Description:        "Top accuracy; slower and pricier but best quality",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              false,
			AdapterType:        AdapterOpenAI,
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.openai.com", Path: "/v1/audio/transcriptions"},
			DocsURL:            docsURL,
		},
		{
			ID:                 "gpt-4o-mini-transcribe",
			Name:               "GPT-4o Mini Transcribe",
			Description:        "Good balance of speed, cost, and quality",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              false,
			AdapterType:        AdapterOpenAI,
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.openai.com", Path: "/v1/audio/transcriptions"},
			DocsURL:            docsURL,
		},
		{
			ID:                 "gpt-4o-realtime-preview",
			Name:               "GPT-4o Realtime Preview",
			Description:        "Instant words as you speak; fastest but most expensive",
			Type:               Transcription,
			SupportsBatch:      false,
			SupportsStreaming:  true,
			Local:              false,
			AdapterType:        AdapterOpenAIRealtime,
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "wss://api.openai.com", Path: "/v1/realtime"},
			DocsURL:            docsURL,
		},
		// LLM models
		{
			ID:                "gpt-4o-mini",
			Name:              "GPT-4o Mini",
			Description:       "Fast and cheap; good default for text cleanup",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenAI,
			Endpoint:          &EndpointConfig{BaseURL: "https://api.openai.com", Path: "/v1/chat/completions"},
		},
		{
			ID:                "gpt-4o",
			Name:              "GPT-4o",
			Description:       "Best quality cleanup; pricier but smarter rewrites",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenAI,
			Endpoint:          &EndpointConfig{BaseURL: "https://api.openai.com", Path: "/v1/chat/completions"},
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
