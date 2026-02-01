package tui

import (
	"fmt"
	"sort"

	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/models/whisper"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// AllProviders is the list of all supported cloud providers (require API keys).
var AllProviders = []string{"openai", "groq", "mistral", "elevenlabs", "deepgram"}

// LocalProviders is the list of local providers (no API key required).
var LocalProviders = []string{"whisper-cpp"}

// providerDisplayNames maps provider IDs to human-readable names.
var providerDisplayNames = map[string]string{
	"openai":      "OpenAI",
	"groq":        "Groq",
	"mistral":     "Mistral",
	"elevenlabs":  "ElevenLabs",
	"deepgram":    "Deepgram",
	"whisper-cpp": "Whisper.cpp (local)",
}

func getProviderDisplayName(providerName string) string {
	if name, ok := providerDisplayNames[providerName]; ok {
		return name
	}
	return providerName
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:7] + "..." + key[len(key)-4:]
}

func hasUserChanges(cfg *config.Config) bool {
	if len(cfg.Providers) > 0 {
		return true
	}
	if cfg.Transcription.APIKey != "" {
		return true
	}
	return false
}

func getConfiguredProviders(cfg *config.Config) []string {
	providers := make([]string, 0, len(cfg.Providers))
	for name, pc := range cfg.Providers {
		if pc.APIKey != "" {
			providers = append(providers, name)
		}
	}
	sort.Strings(providers)
	return providers
}

func isProviderConfigured(cfg *config.Config, providerName string) bool {
	if pc, ok := cfg.Providers[providerName]; ok {
		return pc.APIKey != ""
	}
	return false
}

func mapConfigProviderToRegistry(configProvider string) string {
	switch configProvider {
	case "groq-transcription", "groq-translation":
		return "groq"
	case "mistral-transcription":
		return "mistral"
	default:
		return configProvider
	}
}

func buildModelLabel(m provider.Model) string {
	label := fmt.Sprintf("%s (%s)", m.Name, m.Description)

	if m.Local && m.LocalInfo != nil {
		label += fmt.Sprintf(" [%s]", m.LocalInfo.Size)
	}

	if m.SupportsBothModes() {
		label += " [batch+streaming]"
	} else if m.SupportsStreaming {
		label += " [streaming]"
	}

	return label
}

func getTranscriptionModelOptions(configProvider string) []modelOption {
	if configProvider == "groq-translation" {
		return []modelOption{{ID: "whisper-large-v3", Label: "whisper-large-v3 (only option)"}}
	}

	registryName := mapConfigProviderToRegistry(configProvider)
	p := provider.GetProvider(registryName)
	if p == nil {
		return []modelOption{}
	}

	models := provider.ModelsOfType(p, provider.Transcription)
	options := make([]modelOption, 0, len(models))
	for _, m := range models {
		label := buildModelLabel(m)
		if m.Local && registryName == "whisper-cpp" {
			if whisper.IsInstalled(m.ID) {
				label = "[x] " + label
			} else {
				label = "[ ] " + label
			}
		}
		options = append(options, modelOption{ID: m.ID, Label: label})
	}

	return options
}
