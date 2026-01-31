package provider

import "strings"

// OpenAIProvider implements Provider for OpenAI services
type OpenAIProvider struct{}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) RequiresAPIKey() bool {
	return true
}

func (p *OpenAIProvider) ValidateAPIKey(key string) bool {
	return strings.HasPrefix(key, "sk-")
}

func (p *OpenAIProvider) SupportsTranscription() bool {
	return true
}

func (p *OpenAIProvider) SupportsLLM() bool {
	return true
}

func (p *OpenAIProvider) DefaultTranscriptionModel() string {
	return "whisper-1"
}

func (p *OpenAIProvider) DefaultLLMModel() string {
	return "gpt-4o-mini"
}

func (p *OpenAIProvider) TranscriptionModels() []string {
	return []string{"whisper-1"}
}

func (p *OpenAIProvider) LLMModels() []string {
	return []string{"gpt-4o-mini", "gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo"}
}
