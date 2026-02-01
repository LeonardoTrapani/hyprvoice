package provider

// DeepgramProvider implements Provider for Deepgram transcription services
type DeepgramProvider struct{}

func (p *DeepgramProvider) Name() string {
	return ProviderDeepgram
}

func (p *DeepgramProvider) RequiresAPIKey() bool {
	return true
}

func (p *DeepgramProvider) ValidateAPIKey(key string) bool {
	// Deepgram API keys are alphanumeric, just check non-empty
	return len(key) > 0
}

func (p *DeepgramProvider) IsLocal() bool {
	return false
}

func (p *DeepgramProvider) Models() []Model {
	// https://developers.deepgram.com/docs/models-languages-overview
	nova3Langs := deepgramNova3Languages
	// https://developers.deepgram.com/docs/models-languages-overview
	nova2Langs := deepgramNova2Languages

	docsURL := "https://developers.deepgram.com/docs/language"

	return []Model{
		{
			ID:                 "nova-3",
			Name:               "Nova-3",
			Description:        "Best accuracy, 40+ languages",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  true,
			Local:              false,
			AdapterType:        AdapterDeepgram,
			SupportedLanguages: nova3Langs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.deepgram.com", Path: "/v1/listen"},
			StreamingEndpoint:  &EndpointConfig{BaseURL: "wss://api.deepgram.com", Path: "/v1/listen"},
			DocsURL:            docsURL,
		},
		{
			ID:                 "nova-2",
			Name:               "Nova-2",
			Description:        "Fast, 30+ languages, filler words",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  true,
			Local:              false,
			AdapterType:        AdapterDeepgram,
			SupportedLanguages: nova2Langs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.deepgram.com", Path: "/v1/listen"},
			StreamingEndpoint:  &EndpointConfig{BaseURL: "wss://api.deepgram.com", Path: "/v1/listen"},
			DocsURL:            docsURL,
		},
	}
}

func (p *DeepgramProvider) DefaultModel(t ModelType) string {
	switch t {
	case Transcription:
		return "nova-3"
	}
	return ""
}
