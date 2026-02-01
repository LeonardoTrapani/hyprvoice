package provider

// MistralProvider implements Provider for Mistral services (transcription only)
type MistralProvider struct{}

func (p *MistralProvider) Name() string {
	return ProviderMistral
}

func (p *MistralProvider) RequiresAPIKey() bool {
	return true
}

func (p *MistralProvider) ValidateAPIKey(key string) bool {
	// Mistral API keys don't have a consistent prefix, just check non-empty
	return len(key) > 0
}

func (p *MistralProvider) IsLocal() bool {
	return false
}

func (p *MistralProvider) Models() []Model {
	// https://docs.mistral.ai/capabilities/audio/
	allLangs := mistralTranscriptionLanguages
	docsURL := "https://docs.mistral.ai/capabilities/audio/"

	return []Model{
		{
			ID:                 "voxtral-mini-latest",
			Name:               "Voxtral Mini Latest",
			Description:        "Latest Voxtral model, best for most uses",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  true,
			Local:              false,
			AdapterType:        AdapterOpenAI,
			StreamingAdapter:   "mistral-streaming", // not yet implemented
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.mistral.ai", Path: "/v1/audio/transcriptions"},
			DocsURL:            docsURL,
		},
		{
			ID:                 "voxtral-mini-2507",
			Name:               "Voxtral Mini 2507",
			Description:        "Stable Voxtral version from July 2025",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  true,
			Local:              false,
			AdapterType:        AdapterOpenAI,
			StreamingAdapter:   "mistral-streaming", // not yet implemented
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.mistral.ai", Path: "/v1/audio/transcriptions"},
			DocsURL:            docsURL,
		},
	}
}

func (p *MistralProvider) DefaultModel(t ModelType) string {
	switch t {
	case Transcription:
		return "voxtral-mini-latest"
	}
	return ""
}
