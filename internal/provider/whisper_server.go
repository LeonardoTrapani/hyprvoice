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
			// "default" is sent as the model field in the multipart request.
			// whisper.cpp server ignores it — the model is chosen at startup via -m.
			// Servers that do use the model field (e.g. faster-whisper-server) will
			// need to set transcription.model to a value their server recognises.
			ID:                 "default",
			Name:               "Default",
			Description:        "Uses whichever model the server was started with",
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
		return "default"
	}
	return ""
}
