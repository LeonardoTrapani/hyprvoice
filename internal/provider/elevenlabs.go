package provider

// ElevenLabsProvider implements Provider for ElevenLabs services (transcription only)
type ElevenLabsProvider struct{}

func (p *ElevenLabsProvider) Name() string {
	return ProviderElevenLabs
}

func (p *ElevenLabsProvider) RequiresAPIKey() bool {
	return true
}

func (p *ElevenLabsProvider) ValidateAPIKey(key string) bool {
	// ElevenLabs API keys don't have a consistent prefix, just check non-empty
	return len(key) > 0
}

func (p *ElevenLabsProvider) IsLocal() bool {
	return false
}

func (p *ElevenLabsProvider) Models() []Model {
	// https://elevenlabs.io/speech-to-text
	allLangs := elevenLabsTranscriptionLanguages
	docsURL := "https://elevenlabs.io/speech-to-text"

	return []Model{
		{
			ID:                 "scribe_v1",
			Name:               "Scribe v1",
			Description:        "90+ languages, best accuracy",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              false,
			AdapterType:        AdapterElevenLabs,
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.elevenlabs.io", Path: "/v1/speech-to-text"},
			DocsURL:            docsURL,
		},
		{
			ID:                 "scribe_v2",
			Name:               "Scribe v2",
			Description:        "Lower latency batch transcription",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              false,
			AdapterType:        AdapterElevenLabs,
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.elevenlabs.io", Path: "/v1/speech-to-text"},
			DocsURL:            docsURL,
		},
		{
			ID:                 "scribe_v2_realtime",
			Name:               "Scribe v2 Realtime",
			Description:        "Real-time streaming, <150ms latency",
			Type:               Transcription,
			SupportsBatch:      false,
			SupportsStreaming:  true,
			Local:              false,
			AdapterType:        AdapterElevenLabsStream,
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "wss://api.elevenlabs.io", Path: "/v1/speech-to-text/realtime"},
			DocsURL:            docsURL,
		},
	}
}

func (p *ElevenLabsProvider) DefaultModel(t ModelType) string {
	switch t {
	case Transcription:
		return "scribe_v1"
	}
	return ""
}
