package tui

import (
	"fmt"
	"sort"
	"strings"

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

func getProviderKeyURL(providerName string) string {
	p := provider.GetProvider(providerName)
	if p == nil {
		return ""
	}
	return p.APIKeyURL()
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:7] + "..." + key[len(key)-4:]
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
	case "groq-transcription":
		return "groq"
	case "mistral-transcription":
		return "mistral"
	default:
		return configProvider
	}
}

func buildModelDesc(m provider.Model) string {
	parts := []string{}
	if m.Description != "" {
		parts = append(parts, m.Description)
	} else if m.Name != "" {
		parts = append(parts, m.Name)
	}

	if m.Local {
		parts = append(parts, "local model")
	}

	if m.SupportsBothModes() {
		parts = append(parts, "batch+streaming")
	} else if m.SupportsStreaming {
		parts = append(parts, "streaming")
	} else {
		parts = append(parts, "batch-only")
	}

	if m.Local && m.LocalInfo != nil && m.LocalInfo.Size != "" {
		parts = append(parts, fmt.Sprintf("size %s", m.LocalInfo.Size))
	}

	if len(parts) == 0 {
		return "Transcription model"
	}
	return strings.Join(parts, " - ")
}

func getTranscriptionModelOptions(configProvider string) []modelOption {
	registryName := mapConfigProviderToRegistry(configProvider)
	p := provider.GetProvider(registryName)
	if p == nil {
		return []modelOption{}
	}

	models := provider.ModelsOfType(p, provider.Transcription)
	options := make([]modelOption, 0, len(models))
	for _, m := range models {
		desc := buildModelDesc(m)
		if m.Local && registryName == "whisper-cpp" {
			if whisper.IsInstalled(m.ID) {
				desc = desc + " - installed"
			} else {
				desc = desc + " - not installed"
			}
		}
		options = append(options, modelOption{ID: m.ID, Title: m.ID, Desc: desc})
	}

	return options
}
