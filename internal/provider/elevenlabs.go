package provider

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

func (p *ElevenLabsProvider) SupportsTranscription() bool {
	return true
}

func (p *ElevenLabsProvider) SupportsLLM() bool {
	return false
}

func (p *ElevenLabsProvider) DefaultTranscriptionModel() string {
	return "scribe_v1"
}

func (p *ElevenLabsProvider) DefaultLLMModel() string {
	return ""
}

func (p *ElevenLabsProvider) TranscriptionModels() []string {
	return []string{"scribe_v1", "scribe_v2"}
}

func (p *ElevenLabsProvider) LLMModels() []string {
	return nil
}
