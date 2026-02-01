package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// getProviderDisplayName returns the display name for a provider
func getProviderDisplayName(providerName string) string {
	if name, ok := providerDisplayNames[providerName]; ok {
		return name
	}
	return providerName
}

// maskAPIKey returns a masked version of an API key for display
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:7] + "..." + key[len(key)-4:]
}

// getConfiguredProviders returns list of providers with API keys
func getConfiguredProviders(cfg *config.Config) []string {
	var providers []string
	for name, pc := range cfg.Providers {
		if pc.APIKey != "" {
			providers = append(providers, name)
		}
	}
	return providers
}

// editProviders handles the providers section edit with submenu
func editProviders(cfg *config.Config, onboarding bool) error {
	exitLabel := "Done"
	if onboarding {
		exitLabel = "Next"
	}

	// track if we should default to "back" (Next) after configuring a provider
	defaultToExit := false

	for {
		var options []huh.Option[string]
		for _, name := range AllProviders {
			options = append(options, huh.NewOption(formatProviderOption(cfg, name), name))
		}
		options = append(options, huh.NewOption(exitLabel, "back"))

		selected := ""
		if defaultToExit {
			selected = "back"
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Provider Settings").
					Description("Select a provider to configure API key").
					Options(options...).
					Value(&selected),
			),
		).WithTheme(getTheme())

		if err := form.Run(); err != nil {
			return err
		}

		if selected == "back" {
			return nil
		}

		apiKey, err := configureSingleProvider(cfg, selected)
		if err != nil {
			continue
		}

		if apiKey != "" {
			if cfg.Providers == nil {
				cfg.Providers = make(map[string]config.ProviderConfig)
			}
			cfg.Providers[selected] = config.ProviderConfig{APIKey: apiKey}
			defaultToExit = true
		}
	}
}

// formatProviderOption formats a provider menu option with status
func formatProviderOption(cfg *config.Config, name string) string {
	var status string
	if pc, exists := cfg.Providers[name]; exists && pc.APIKey != "" {
		status = "(configured)"
	} else {
		status = "(not configured)"
	}

	switch name {
	case "openai":
		return fmt.Sprintf("OpenAI - Whisper + GPT %s", status)
	case "groq":
		return fmt.Sprintf("Groq - Whisper + Llama %s", status)
	case "mistral":
		return fmt.Sprintf("Mistral - Voxtral %s", status)
	case "elevenlabs":
		return fmt.Sprintf("ElevenLabs - Scribe %s", status)
	default:
		return fmt.Sprintf("%s %s", name, status)
	}
}

// configureSingleProvider handles the complete flow for configuring a single provider's API key.
// Shows confirm dialog if key exists, then prompts for new key if needed.
// Returns the new API key (empty if user kept current) and any error.
func configureSingleProvider(cfg *config.Config, providerName string) (string, error) {
	var existingKey string
	if pc, exists := cfg.Providers[providerName]; exists && pc.APIKey != "" {
		existingKey = pc.APIKey
	}

	if existingKey != "" {
		displayName := getProviderDisplayName(providerName)
		masked := maskAPIKey(existingKey)

		var update bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("%s API Key", displayName)).
					Description(fmt.Sprintf("Current: %s", masked)).
					Affirmative("Update key").
					Negative("Keep current").
					Value(&update),
			),
		).WithTheme(getTheme())

		if err := confirmForm.Run(); err != nil {
			return "", err
		}

		if !update {
			return "", nil
		}
	}

	return inputAPIKey(providerName)
}

func inputAPIKey(providerName string) (string, error) {
	p := provider.GetProvider(providerName)
	displayName := getProviderDisplayName(providerName)
	if p != nil {
		if name, ok := providerDisplayNames[p.Name()]; ok {
			displayName = name
		}
	}

	var apiKey string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("%s API Key", displayName)).
				Description(fmt.Sprintf("Enter your %s API key", displayName)).
				EchoMode(huh.EchoModePassword).
				Value(&apiKey).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("API key is required")
					}
					if p != nil && !p.ValidateAPIKey(s) {
						return fmt.Errorf("invalid API key format for %s", displayName)
					}
					return nil
				}),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return "", err
	}

	return apiKey, nil
}

// ensureProviderConfigured prompts for API key if provider not configured
func ensureProviderConfigured(cfg *config.Config, selectedProvider string, configuredProviders []string) []string {
	providerName := selectedProvider
	switch selectedProvider {
	case "groq-transcription", "groq-translation":
		providerName = "groq"
	case "mistral-transcription":
		providerName = "mistral"
	}

	for _, p := range configuredProviders {
		if p == providerName {
			return configuredProviders
		}
	}

	apiKey, err := configureSingleProvider(cfg, providerName)
	if err != nil || apiKey == "" {
		return configuredProviders
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}
	cfg.Providers[providerName] = config.ProviderConfig{APIKey: apiKey}

	return append(configuredProviders, providerName)
}
