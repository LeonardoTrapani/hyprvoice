package provider

// Provider defines the interface for a transcription/LLM service provider
type Provider interface {
	Name() string
	RequiresAPIKey() bool
	ValidateAPIKey(key string) bool
	SupportsTranscription() bool
	SupportsLLM() bool
	DefaultTranscriptionModel() string
	DefaultLLMModel() string
	TranscriptionModels() []string
	LLMModels() []string
}

// ProviderConfig holds configuration for a single provider
type ProviderConfig struct {
	APIKey string `toml:"api_key"`
}

var registry = make(map[string]Provider)

func init() {
	Register(&OpenAIProvider{})
	Register(&GroqProvider{})
	Register(&MistralProvider{})
	Register(&ElevenLabsProvider{})
}

// Register adds a provider to the registry
func Register(p Provider) {
	registry[p.Name()] = p
}

// GetProvider returns a provider by name, or nil if not found
func GetProvider(name string) Provider {
	return registry[name]
}

// ListProviders returns all registered provider names
func ListProviders() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// ListProvidersWithTranscription returns providers that support transcription
func ListProvidersWithTranscription() []string {
	var names []string
	for name, p := range registry {
		if p.SupportsTranscription() {
			names = append(names, name)
		}
	}
	return names
}

// ListProvidersWithLLM returns providers that support LLM
func ListProvidersWithLLM() []string {
	var names []string
	for name, p := range registry {
		if p.SupportsLLM() {
			names = append(names, name)
		}
	}
	return names
}
