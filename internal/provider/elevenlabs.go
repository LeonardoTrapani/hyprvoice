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
	allLangs := language.AllLanguageCodes()

	return []Model{
		{
			ID:                 "scribe_v1",
			Name:               "Scribe v1",
			Description:        "99 languages, best accuracy",
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
	}
}

func (p *ElevenLabsProvider) DefaultModel(t ModelType) string {
	switch t {
	case Transcription:
		return "scribe_v1"
	}
	return ""
}
