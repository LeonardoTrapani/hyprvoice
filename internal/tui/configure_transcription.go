package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/deps"
	"github.com/leonardotrapani/hyprvoice/internal/models/whisper"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// editTranscription handles the transcription section edit with smart provider detection
func editTranscription(cfg *config.Config, configuredProviders []string) ([]string, error) {
	var transcriptionOptions []huh.Option[string]

	// add local provider first (whisper-cpp)
	whisperStatus := deps.CheckWhisperCli()
	if whisperStatus.Installed {
		transcriptionOptions = append(transcriptionOptions,
			huh.NewOption("Whisper.cpp (local, no API key)", "whisper-cpp"))
	} else {
		transcriptionOptions = append(transcriptionOptions,
			huh.NewOption("Whisper.cpp (whisper-cli not found)", "whisper-cpp-disabled"))
	}

	// add configured cloud providers
	for _, name := range configuredProviders {
		p := provider.GetProvider(name)
		if p != nil && len(provider.ModelsOfType(p, provider.Transcription)) > 0 {
			switch name {
			case "openai":
				transcriptionOptions = append(transcriptionOptions,
					huh.NewOption("OpenAI Whisper", "openai"))
			case "groq":
				transcriptionOptions = append(transcriptionOptions,
					huh.NewOption("Groq Whisper", "groq-transcription"))
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

	// handle disabled whisper-cpp selection
	if selectedProvider == "whisper-cpp-disabled" {
		fmt.Println()
		fmt.Println(StyleWarning.Render("whisper-cli not found in PATH"))
		fmt.Println(StyleMuted.Render("Install whisper.cpp to use local transcription:"))
		fmt.Println(StyleMuted.Render("  https://github.com/ggerganov/whisper.cpp"))
		fmt.Println()

		var proceed bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Continue?").
					Affirmative("Choose another provider").
					Negative("Cancel").
					Value(&proceed),
			),
		).WithTheme(getTheme())
		if err := form.Run(); err != nil {
			return configuredProviders, err
		}
		if proceed {
			return editTranscription(cfg, configuredProviders)
		}
		return configuredProviders, nil
	}

	// local providers don't need API key configuration
	if selectedProvider != "whisper-cpp" {
		configuredProviders = ensureProviderConfigured(cfg, selectedProvider, configuredProviders)
	}
	cfg.Transcription.Provider = selectedProvider

	modelOptions := getTranscriptionModelOptions(selectedProvider)
	selectedModel := cfg.Transcription.Model
	if selectedModel == "" && len(modelOptions) > 0 {
		// skip header options (empty value) to find first real model
		for _, opt := range modelOptions {
			if opt.Value != "" {
				selectedModel = opt.Value
				break
			}
		}
	}

	modelDesc := ""
	if cfg.Transcription.Model != "" {
		modelDesc = fmt.Sprintf("Currently: %s", cfg.Transcription.Model)
	}

	modelForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Transcription Model").
				Description(modelDesc).
				Options(modelOptions...).
				Value(&selectedModel),
		),
	).WithTheme(getTheme())

	if err := modelForm.Run(); err != nil {
		return configuredProviders, err
	}

	// if user selected a section header (empty value), re-prompt
	if selectedModel == "" {
		return editTranscription(cfg, configuredProviders)
	}

	registryName := mapConfigProviderToRegistry(selectedProvider)

	// for whisper-cpp, check if model needs download
	if selectedProvider == "whisper-cpp" && !whisper.IsInstalled(selectedModel) {
		modelInfo := whisper.GetModel(selectedModel)
		if modelInfo == nil {
			return configuredProviders, fmt.Errorf("unknown model: %s", selectedModel)
		}

		var confirm bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Download %s (%s)?", modelInfo.Name, modelInfo.Size)).
					Description("Model is not installed. Download now?").
					Affirmative("Download").
					Negative("Cancel").
					Value(&confirm),
			),
		).WithTheme(getTheme())

		if err := confirmForm.Run(); err != nil {
			return configuredProviders, err
		}

		if !confirm {
			return configuredProviders, nil
		}

		// download with progress
		fmt.Println()
		fmt.Printf("Downloading %s...\n", modelInfo.Name)

		lastPct := 0
		err := whisper.Download(context.Background(), selectedModel, func(downloaded, total int64) {
			if total > 0 {
				pct := int(downloaded * 100 / total)
				if pct >= lastPct+10 {
					fmt.Printf("  %d%%\n", pct)
					lastPct = pct
				}
			}
		})

		if err != nil {
			fmt.Println(StyleError.Render(fmt.Sprintf("Download failed: %v", err)))
			return configuredProviders, err
		}

		fmt.Println(StyleSuccess.Render(fmt.Sprintf("Downloaded %s", modelInfo.Name)))
		fmt.Println()
	}

	cfg.Transcription.Model = selectedModel

	// select language for this model
	model, err := provider.GetModel(registryName, selectedModel)
	if err != nil {
		return configuredProviders, err
	}

	if cfg.Transcription.Language != "" && !model.SupportsLanguage(cfg.Transcription.Language) {
		cfg.Transcription.Language = ""
	}

	if len(model.SupportedLanguages) <= 1 {
		if len(model.SupportedLanguages) == 1 {
			cfg.Transcription.Language = model.SupportedLanguages[0]
		} else {
			cfg.Transcription.Language = ""
		}
	} else {
		languageOptions := getModelLanguageOptions(model, cfg.Transcription.Language)
		selectedLanguage := cfg.Transcription.Language

		languageForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Language").
					Description("Select language for transcription").
					Options(languageOptions...).
					Filtering(true).
					Value(&selectedLanguage),
			),
		).WithTheme(getTheme())

		if err := languageForm.Run(); err != nil {
			return configuredProviders, err
		}

		cfg.Transcription.Language = selectedLanguage
	}

	// set streaming mode based on model capabilities
	if model.SupportsBothModes() {
		useStreaming := cfg.Transcription.Streaming
		streamingForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Enable streaming mode?").
					Description("This model supports both batch and streaming modes").
					Affirmative("Yes, use streaming (real-time)").
					Negative("No, use batch (after recording)").
					Value(&useStreaming),
			),
		).WithTheme(getTheme())

		if err := streamingForm.Run(); err != nil {
			return configuredProviders, err
		}
		cfg.Transcription.Streaming = useStreaming
	} else if model.SupportsStreaming {
		cfg.Transcription.Streaming = true
		fmt.Println(StyleSuccess.Render("Streaming mode enabled (this model only supports streaming)"))
	} else {
		cfg.Transcription.Streaming = false
	}

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
			huh.NewOption("Groq Whisper (not configured)", "groq-transcription"))
	}
	if !configured["mistral"] {
		options = append(options, huh.NewOption("Mistral Voxtral (not configured)", "mistral-transcription"))
	}
	if !configured["elevenlabs"] {
		options = append(options, huh.NewOption("ElevenLabs Scribe (not configured)", "elevenlabs"))
	}
	return options
}

func getTranscriptionModelOptions(configProvider string) []huh.Option[string] {
	// map config provider name to registry provider name
	registryName := mapConfigProviderToRegistry(configProvider)
	p := provider.GetProvider(registryName)
	if p == nil {
		return []huh.Option[string]{}
	}

	models := provider.ModelsOfType(p, provider.Transcription)

	var options []huh.Option[string]
	for _, m := range models {
		label := buildModelLabel(m)
		if m.Local && registryName == "whisper-cpp" {
			if whisper.IsInstalled(m.ID) {
				label = "[x] " + label
			} else {
				label = "[ ] " + label
			}
		}
		options = append(options, huh.NewOption(label, m.ID))
	}

	return options
}

// mapConfigProviderToRegistry maps config provider names to registry provider names
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

// buildModelLabel creates the display label for a model option
func buildModelLabel(m provider.Model) string {
	label := fmt.Sprintf("%s (%s)", m.Name, m.Description)

	// append size for local models
	if m.Local && m.LocalInfo != nil {
		label += fmt.Sprintf(" [%s]", m.LocalInfo.Size)
	}

	// append mode capabilities
	if m.SupportsBothModes() {
		label += " [batch+streaming]"
	} else if m.SupportsStreaming {
		label += " [streaming]"
	}

	return label
}
