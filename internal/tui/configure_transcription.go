package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/language"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// editTranscription handles the transcription section edit with smart provider detection
func editTranscription(cfg *config.Config, configuredProviders []string) ([]string, error) {
	var transcriptionOptions []huh.Option[string]
	for _, name := range configuredProviders {
		p := provider.GetProvider(name)
		if p != nil && len(provider.ModelsOfType(p, provider.Transcription)) > 0 {
			switch name {
			case "openai":
				transcriptionOptions = append(transcriptionOptions,
					huh.NewOption("OpenAI Whisper", "openai"))
			case "groq":
				transcriptionOptions = append(transcriptionOptions,
					huh.NewOption("Groq Whisper (transcription)", "groq-transcription"),
					huh.NewOption("Groq Whisper (translate to English)", "groq-translation"))
			case "mistral":
				transcriptionOptions = append(transcriptionOptions,
					huh.NewOption("Mistral Voxtral", "mistral-transcription"))
			case "elevenlabs":
				transcriptionOptions = append(transcriptionOptions,
					huh.NewOption("ElevenLabs Scribe", "elevenlabs"))
			}
		}
	}

	unconfiguredOptions := getUnconfiguredTranscriptionOptions(configuredProviders)
	if len(unconfiguredOptions) > 0 {
		transcriptionOptions = append(transcriptionOptions, unconfiguredOptions...)
	}

	if len(transcriptionOptions) == 0 {
		return configuredProviders, fmt.Errorf("no transcription providers available")
	}

	selectedProvider := cfg.Transcription.Provider
	if selectedProvider == "" && len(transcriptionOptions) > 0 {
		selectedProvider = transcriptionOptions[0].Value
	}

	providerDesc := "Choose which service to use for speech-to-text"
	if cfg.Transcription.Provider != "" {
		providerDesc = fmt.Sprintf("Currently: %s/%s", cfg.Transcription.Provider, cfg.Transcription.Model)
	}

	providerForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Transcription Provider").
				Description(providerDesc).
				Options(transcriptionOptions...).
				Value(&selectedProvider),
		),
	).WithTheme(getTheme())

	if err := providerForm.Run(); err != nil {
		return configuredProviders, err
	}

	configuredProviders = ensureProviderConfigured(cfg, selectedProvider, configuredProviders)
	cfg.Transcription.Provider = selectedProvider

	modelOptions := getTranscriptionModelOptions(selectedProvider, cfg.Transcription.Language)
	selectedModel := cfg.Transcription.Model
	if selectedModel == "" && len(modelOptions) > 0 {
		selectedModel = modelOptions[0].Value
	}

	modelDesc := ""
	if cfg.Transcription.Model != "" {
		modelDesc = fmt.Sprintf("Currently: %s", cfg.Transcription.Model)
	}

	language := cfg.Transcription.Language

	langDesc := "ISO-639-1 code (e.g., 'en', 'es', 'fr') or empty for auto-detect"
	if cfg.Transcription.Language != "" {
		langDesc = fmt.Sprintf("Currently: %s. %s", cfg.Transcription.Language, langDesc)
	}

	modelForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Transcription Model").
				Description(modelDesc).
				Options(modelOptions...).
				Value(&selectedModel),
			huh.NewInput().
				Title("Language").
				Description(langDesc).
				Placeholder("auto-detect").
				Value(&language),
		),
	).WithTheme(getTheme())

	if err := modelForm.Run(); err != nil {
		return configuredProviders, err
	}

	cfg.Transcription.Model = selectedModel
	cfg.Transcription.Language = language

	return configuredProviders, nil
}

// getUnconfiguredTranscriptionOptions returns options for providers not yet configured
func getUnconfiguredTranscriptionOptions(configuredProviders []string) []huh.Option[string] {
	configured := make(map[string]bool)
	for _, p := range configuredProviders {
		configured[p] = true
	}

	var options []huh.Option[string]
	if !configured["openai"] {
		options = append(options, huh.NewOption("OpenAI Whisper (not configured)", "openai"))
	}
	if !configured["groq"] {
		options = append(options,
			huh.NewOption("Groq Whisper transcription (not configured)", "groq-transcription"),
			huh.NewOption("Groq Whisper translation (not configured)", "groq-translation"))
	}
	if !configured["mistral"] {
		options = append(options, huh.NewOption("Mistral Voxtral (not configured)", "mistral-transcription"))
	}
	if !configured["elevenlabs"] {
		options = append(options, huh.NewOption("ElevenLabs Scribe (not configured)", "elevenlabs"))
	}
	return options
}

func getTranscriptionModelOptions(configProvider string, currentLang string) []huh.Option[string] {
	// special case: groq-translation only supports whisper-large-v3
	if configProvider == "groq-translation" {
		return []huh.Option[string]{
			huh.NewOption("whisper-large-v3 (only option)", "whisper-large-v3"),
		}
	}

	// map config provider name to registry provider name
	registryName := mapConfigProviderToRegistry(configProvider)
	p := provider.GetProvider(registryName)
	if p == nil {
		return []huh.Option[string]{}
	}

	models := provider.ModelsOfType(p, provider.Transcription)
	var options []huh.Option[string]

	for _, m := range models {
		// skip streaming models for now (not yet implemented)
		if m.Streaming {
			continue
		}

		label := buildModelLabel(m, currentLang)
		options = append(options, huh.NewOption(label, m.ID))
	}

	return options
}

// mapConfigProviderToRegistry maps config provider names to registry provider names
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

// buildModelLabel creates the display label for a model option
func buildModelLabel(m provider.Model, currentLang string) string {
	label := fmt.Sprintf("%s (%s)", m.Name, m.Description)

	// append size for local models
	if m.Local && m.LocalInfo != nil {
		label += fmt.Sprintf(" [%s]", m.LocalInfo.Size)
	}

	// append streaming tag
	if m.Streaming {
		label += " [streaming]"
	}

	// append language warning if model doesn't support current language
	if currentLang != "" && !m.SupportsLanguage(currentLang) {
		langName := getLangName(currentLang)
		label += fmt.Sprintf(" (does not support %s)", langName)
	}

	return label
}

// getLangName returns a human-readable language name for a code
func getLangName(code string) string {
	lang := language.FromCode(code)
	if lang.Code == "" {
		return code // unknown code, return as-is
	}
	return lang.Name
}
