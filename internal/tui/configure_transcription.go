package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/deps"
	"github.com/leonardotrapani/hyprvoice/internal/language"
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

	// use effective language for model compatibility display
	effectiveLanguage := cfg.General.Language
	if cfg.Transcription.Language != "" {
		effectiveLanguage = cfg.Transcription.Language
	}

	modelOptions := getTranscriptionModelOptions(selectedProvider, effectiveLanguage)
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

	// validate language-model compatibility before saving
	registryName := mapConfigProviderToRegistry(selectedProvider)
	if err := provider.ValidateModelLanguage(registryName, selectedModel, effectiveLanguage); err != nil {
		// show error dialog - user needs to change language in Language menu
		fmt.Println()
		fmt.Println(StyleError.Render("Language-Model Incompatibility"))
		fmt.Println(StyleMuted.Render(err.Error()))
		fmt.Println()
		fmt.Println(StyleMuted.Render("You can:"))
		fmt.Println(StyleMuted.Render("  - Choose a different model that supports your language"))
		fmt.Println(StyleMuted.Render("  - Change language to 'Auto-detect' in the Language menu"))
		fmt.Println()

		var retry bool
		retryForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Try again?").
					Description("Choose a different model").
					Affirmative("Yes, let me pick another model").
					Negative("Cancel").
					Value(&retry),
			),
		).WithTheme(getTheme())

		if err := retryForm.Run(); err != nil {
			return configuredProviders, err
		}

		if retry {
			// recurse to let user pick another model
			return editTranscription(cfg, configuredProviders)
		}
		return configuredProviders, nil
	}

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

	// set streaming mode based on model capabilities
	model, err := provider.GetModel(registryName, selectedModel)
	if err == nil {
		if model.SupportsBothModes() {
			// model supports both: ask user
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
			// streaming-only model
			cfg.Transcription.Streaming = true
			fmt.Println(StyleSuccess.Render("Streaming mode enabled (this model only supports streaming)"))
		} else {
			// batch-only model
			cfg.Transcription.Streaming = false
		}
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
		label := buildModelLabel(m, currentLang)
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

	// append mode capabilities
	if m.SupportsBothModes() {
		label += " [batch+streaming]"
	} else if m.SupportsStreaming {
		label += " [streaming]"
	}
	// batch-only models don't need a tag (it's the default)

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
