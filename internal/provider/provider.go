package provider

import (
	"errors"
	"fmt"
	"strings"
)

// Provider defines the interface for a transcription/LLM service provider
type Provider interface {
	Name() string
	RequiresAPIKey() bool
	ValidateAPIKey(key string) bool
	IsLocal() bool
	Models() []Model
	DefaultModel(t ModelType) string
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
	Register(&WhisperCppProvider{})
	Register(&DeepgramProvider{})
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
		if hasModelsOfType(p, Transcription) {
			names = append(names, name)
		}
	}
	return names
}

// ListProvidersWithLLM returns providers that support LLM
func ListProvidersWithLLM() []string {
	var names []string
	for name, p := range registry {
		if hasModelsOfType(p, LLM) {
			names = append(names, name)
		}
	}
	return names
}

// hasModelsOfType returns true if provider has any models of the given type
func hasModelsOfType(p Provider, t ModelType) bool {
	for _, m := range p.Models() {
		if m.Type == t {
			return true
		}
	}
	return false
}

// GetModel returns a model from a specific provider, or error if not found
func GetModel(providerName, modelID string) (*Model, error) {
	p := GetProvider(providerName)
	if p == nil {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	for _, m := range p.Models() {
		if m.ID == modelID {
			return &m, nil
		}
	}
	return nil, fmt.Errorf("model %s not found in provider %s", modelID, providerName)
}

// ModelsOfType returns all models of the given type from a provider
func ModelsOfType(p Provider, t ModelType) []Model {
	var result []Model
	for _, m := range p.Models() {
		if m.Type == t {
			result = append(result, m)
		}
	}
	return result
}

// FindModelByID searches all providers for a model with the given ID
func FindModelByID(modelID string) (*Model, Provider, error) {
	for _, p := range registry {
		for _, m := range p.Models() {
			if m.ID == modelID {
				return &m, p, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("model %s not found in any provider", modelID)
}

// ModelsForLanguage returns models from a provider that support the given language
func ModelsForLanguage(p Provider, t ModelType, langCode string) []Model {
	var result []Model
	for _, m := range p.Models() {
		if m.Type == t && m.SupportsLanguage(langCode) {
			result = append(result, m)
		}
	}
	return result
}

// ValidateModelLanguage checks if a model supports the given language.
// Returns error with list of supported languages if not supported.
// Returns nil if langCode is "" (auto) - auto is always supported.
func ValidateModelLanguage(providerName, modelID, langCode string) error {
	if langCode == "" {
		return nil // auto always supported
	}

	model, err := GetModel(providerName, modelID)
	if err != nil {
		return err
	}

	if model.SupportsLanguage(langCode) {
		return nil
	}

	// truncate supported languages list for error message
	supported := model.SupportedLanguages
	suffix := ""
	if len(supported) > 5 {
		supported = supported[:5]
		suffix = "..."
	}

	// build error with docs URL if available
	docsHint := ""
	if model.DocsURL != "" {
		docsHint = fmt.Sprintf(" See %s for full list.", model.DocsURL)
	}

	langLabel := LanguageLabel(langCode)
	if langLabel == "" {
		langLabel = fmt.Sprintf("language '%s'", langCode)
	}

	return fmt.Errorf(
		"model %s does not support %s.%s Supported: %s%s",
		model.Name,
		langLabel,
		docsHint,
		strings.Join(supported, ", "),
		suffix,
	)
}

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrModelNotFound    = errors.New("model not found")
)
