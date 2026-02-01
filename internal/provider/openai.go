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
			Description:        "OpenAI's production speech-to-text model",
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
			Description:        "High quality transcription with GPT-4o",
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
			Description:        "Fast transcription with GPT-4o Mini",
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
			Description:        "Real-time streaming transcription with GPT-4o",
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
			Description:       "Fast and affordable GPT-4 variant",
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
			Description:       "Most capable GPT-4 model",
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
