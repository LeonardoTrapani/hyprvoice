package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/config"
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

	modelOptions := getTranscriptionModelOptions(selectedProvider)
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

func getTranscriptionModelOptions(provider string) []huh.Option[string] {
	switch provider {
	case "openai":
		return []huh.Option[string]{
			huh.NewOption("whisper-1", "whisper-1"),
		}
	case "groq-transcription":
		return []huh.Option[string]{
			huh.NewOption("whisper-large-v3-turbo (faster)", "whisper-large-v3-turbo"),
			huh.NewOption("whisper-large-v3 (standard)", "whisper-large-v3"),
		}
	case "groq-translation":
		return []huh.Option[string]{
			huh.NewOption("whisper-large-v3 (only option)", "whisper-large-v3"),
		}
	case "mistral-transcription":
		return []huh.Option[string]{
			huh.NewOption("voxtral-mini-latest (recommended)", "voxtral-mini-latest"),
			huh.NewOption("voxtral-mini-2507", "voxtral-mini-2507"),
		}
	case "elevenlabs":
		return []huh.Option[string]{
			huh.NewOption("scribe_v1 (99 languages, best accuracy)", "scribe_v1"),
			huh.NewOption("scribe_v2 (real-time, lower latency)", "scribe_v2"),
		}
	default:
		return []huh.Option[string]{}
	}
}
