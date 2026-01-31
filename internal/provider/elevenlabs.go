package provider

import "github.com/leonardotrapani/hyprvoice/internal/language"

// ElevenLabsProvider implements Provider for ElevenLabs services (transcription only)
type ElevenLabsProvider struct{}

func (p *ElevenLabsProvider) Name() string {
	return "elevenlabs"
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
	// ElevenLabs Scribe supports 90+ languages, including all 57 from our master list
	// See: https://elevenlabs.io/speech-to-text
	allLangs := language.AllLanguageCodes()

	return []Model{
		// batch models
		{
			ID:                 "scribe_v1",
			Name:               "Scribe v1",
			Description:        "90+ languages, best accuracy",
			Type:               Transcription,
			Streaming:          false,
			Local:              false,
			AdapterType:        "elevenlabs",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.elevenlabs.io", Path: "/v1/speech-to-text"},
		},
		{
			ID:                 "scribe_v2",
			Name:               "Scribe v2",
			Description:        "Lower latency, real-time optimized",
			Type:               Transcription,
			Streaming:          false,
			Local:              false,
			AdapterType:        "elevenlabs",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "https://api.elevenlabs.io", Path: "/v1/speech-to-text"},
		},
		// streaming models
		{
			ID:                 "scribe_v1-streaming",
			Name:               "Scribe v1 Streaming",
			Description:        "Real-time transcription, 90+ languages",
			Type:               Transcription,
			Streaming:          true,
			Local:              false,
			AdapterType:        "elevenlabs-streaming",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "wss://api.elevenlabs.io", Path: "/v1/speech-to-text/realtime"},
		},
		{
			ID:                 "scribe_v2-streaming",
			Name:               "Scribe v2 Streaming",
			Description:        "Real-time with <150ms latency",
			Type:               Transcription,
			Streaming:          true,
			Local:              false,
			AdapterType:        "elevenlabs-streaming",
			SupportedLanguages: allLangs,
			Endpoint:           &EndpointConfig{BaseURL: "wss://api.elevenlabs.io", Path: "/v1/speech-to-text/realtime"},
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
