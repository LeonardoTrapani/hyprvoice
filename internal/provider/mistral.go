package provider

// MistralProvider implements Provider for Mistral services (transcription only)
type MistralProvider struct{}

func (p *MistralProvider) Name() string {
	return "mistral"
}

func (p *MistralProvider) RequiresAPIKey() bool {
	return true
}

func (p *MistralProvider) ValidateAPIKey(key string) bool {
	// Mistral API keys don't have a consistent prefix, just check non-empty
	return len(key) > 0
}

func (p *MistralProvider) SupportsTranscription() bool {
	return true
}

func (p *MistralProvider) SupportsLLM() bool {
	return false
}

func (p *MistralProvider) DefaultTranscriptionModel() string {
	return "voxtral-mini-latest"
}

func (p *MistralProvider) DefaultLLMModel() string {
	return ""
}

func (p *MistralProvider) TranscriptionModels() []string {
	return []string{"voxtral-mini-latest", "voxtral-mini-2507"}
}

func (p *MistralProvider) LLMModels() []string {
	return nil
}
