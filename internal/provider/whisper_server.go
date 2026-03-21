package provider

// WhisperServerProvider implements Provider for a local whisper.cpp HTTP server
// using the native /inference endpoint
type WhisperServerProvider struct{}

func (p *WhisperServerProvider) Name() string {
	return ProviderWhisperServer
}

func (p *WhisperServerProvider) RequiresAPIKey() bool {
	return false
}

func (p *WhisperServerProvider) ValidateAPIKey(key string) bool {
	return true // API key is optional for local servers
}

func (p *WhisperServerProvider) APIKeyURL() string {
	return ""
}

func (p *WhisperServerProvider) IsLocal() bool {
	return true
}

func (p *WhisperServerProvider) Models() []Model {
	allLangs := whisperTranscriptionLanguages
	docsURL := "https://github.com/ggml-org/whisper.cpp/tree/master/examples/server"
	defaultEndpoint := &EndpointConfig{BaseURL: "http://localhost:8080", Path: "/inference"}

	return []Model{
		{
			ID:                 "whisper-1",
			Name:               "Whisper 1",
			Description:        "Default; model is selected by the server at startup",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              true,
			AdapterType:        AdapterWhisperServer,
			SupportedLanguages: allLangs,
			Endpoint:           defaultEndpoint,
			DocsURL:            docsURL,
		},
		{
			ID:                 "whisper-large-v3",
			Name:               "Whisper Large v3",
			Description:        "Best accuracy; use with a server loaded with the large-v3 model",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              true,
			AdapterType:        AdapterWhisperServer,
			SupportedLanguages: allLangs,
			Endpoint:           defaultEndpoint,
			DocsURL:            docsURL,
		},
		{
			ID:                 "whisper-large-v3-turbo",
			Name:               "Whisper Large v3 Turbo",
			Description:        "Near-best accuracy with better speed; use with a server loaded with the large-v3-turbo model",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              true,
			AdapterType:        AdapterWhisperServer,
			SupportedLanguages: allLangs,
			Endpoint:           defaultEndpoint,
			DocsURL:            docsURL,
		},
	}
}

func (p *WhisperServerProvider) DefaultModel(t ModelType) string {
	switch t {
	case Transcription:
		return "whisper-1" // placeholder; whisper.cpp server ignores the model field — model is chosen at startup
	}
	return ""
}
