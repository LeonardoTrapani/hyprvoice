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

func (p *MistralProvider) APIKeyURL() string {
	return "https://admin.mistral.ai/organization/api-keys"
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
			Description:        "EU-hosted; good for data residency or Mistral ecosystem",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              false,
			AdapterType:        AdapterOpenAI,
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
