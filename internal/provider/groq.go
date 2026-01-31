package provider

import "strings"

// GroqProvider implements Provider for Groq services
type GroqProvider struct{}

func (p *GroqProvider) Name() string {
	return "groq"
}

func (p *GroqProvider) RequiresAPIKey() bool {
	return true
}

func (p *GroqProvider) ValidateAPIKey(key string) bool {
	return strings.HasPrefix(key, "gsk_")
}

func (p *GroqProvider) SupportsTranscription() bool {
	return true
}

func (p *GroqProvider) SupportsLLM() bool {
	return true
}

func (p *GroqProvider) DefaultTranscriptionModel() string {
	return "whisper-large-v3-turbo"
}

func (p *GroqProvider) DefaultLLMModel() string {
	return "llama-3.3-70b-versatile"
}

func (p *GroqProvider) TranscriptionModels() []string {
	return []string{"whisper-large-v3", "whisper-large-v3-turbo"}
}

func (p *GroqProvider) LLMModels() []string {
	return []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768"}
}
