package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// editLLM handles the LLM section edit with smart provider detection
func editLLM(cfg *config.Config, configuredProviders []string) ([]string, error) {
	var llmProviders []string
	for _, name := range configuredProviders {
		p := provider.GetProvider(name)
		if p != nil && len(provider.ModelsOfType(p, provider.LLM)) > 0 {
			llmProviders = append(llmProviders, name)
		}
	}

	postProcessing := cfg.LLM.PostProcessing
	if !postProcessing.RemoveStutters && !postProcessing.AddPunctuation &&
		!postProcessing.FixGrammar && !postProcessing.RemoveFillerWords {
		postProcessing = config.LLMPostProcessingConfig{
			RemoveStutters:    true,
			AddPunctuation:    true,
			FixGrammar:        true,
			RemoveFillerWords: true,
		}
	}
	customPrompt := cfg.LLM.CustomPrompt

	enableLLM := cfg.LLM.Enabled

	enableDesc := "LLM improves transcription by fixing grammar, removing stutters, and cleaning up text"
	if cfg.LLM.Enabled {
		enableDesc = fmt.Sprintf("Currently: enabled (%s/%s). %s", cfg.LLM.Provider, cfg.LLM.Model, enableDesc)
	} else {
		enableDesc = "Currently: disabled. " + enableDesc
	}

	enableForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable LLM Post-Processing? (Recommended)").
				Description(enableDesc).
				Affirmative("Yes (Recommended)").
				Negative("No").
				Value(&enableLLM),
		),
	).WithTheme(getTheme())

	if err := enableForm.Run(); err != nil {
		return configuredProviders, err
	}

	if !enableLLM {
		cfg.LLM.Enabled = false
		return configuredProviders, nil
	}

	var llmOptions []huh.Option[string]
	for _, name := range llmProviders {
		switch name {
		case "openai":
			llmOptions = append(llmOptions, huh.NewOption("OpenAI GPT", "openai"))
		case "groq":
			llmOptions = append(llmOptions, huh.NewOption("Groq Llama (fast)", "groq"))
		}
	}

	unconfiguredLLM := getUnconfiguredLLMOptions(configuredProviders)
	if len(unconfiguredLLM) > 0 {
		llmOptions = append(llmOptions, unconfiguredLLM...)
	}

	if len(llmOptions) == 0 {
		fmt.Println(StyleError.Render("No LLM providers available. Please configure OpenAI or Groq first."))
		cfg.LLM.Enabled = false
		return configuredProviders, nil
	}

	selectedProvider := cfg.LLM.Provider
	if selectedProvider == "" && len(llmOptions) > 0 {
		selectedProvider = llmOptions[0].Value
	}

	llmProviderDesc := "Choose which service to use for text post-processing"
	if cfg.LLM.Provider != "" {
		llmProviderDesc = fmt.Sprintf("Currently: %s/%s", cfg.LLM.Provider, cfg.LLM.Model)
	}

	providerForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("LLM Provider").
				Description(llmProviderDesc).
				Options(llmOptions...).
				Value(&selectedProvider),
		),
	).WithTheme(getTheme())

	if err := providerForm.Run(); err != nil {
		return configuredProviders, err
	}

	configuredProviders = ensureProviderConfigured(cfg, selectedProvider, configuredProviders)
	cfg.LLM.Provider = selectedProvider

	modelOptions := getLLMModelOptions(selectedProvider)
	selectedModel := cfg.LLM.Model
	if selectedModel == "" && len(modelOptions) > 0 {
		selectedModel = modelOptions[0].Value
	}

	llmModelDesc := ""
	if cfg.LLM.Model != "" {
		llmModelDesc = fmt.Sprintf("Currently: %s", cfg.LLM.Model)
	}

	modelForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("LLM Model").
				Description(llmModelDesc).
				Options(modelOptions...).
				Value(&selectedModel),
		),
	).WithTheme(getTheme())

	if err := modelForm.Run(); err != nil {
		return configuredProviders, err
	}

	cfg.LLM.Model = selectedModel

	var ppErr error
	postProcessing, ppErr = selectPostProcessingOptions(postProcessing)
	if ppErr != nil {
		return configuredProviders, ppErr
	}

	cfg.LLM.PostProcessing = postProcessing

	enableCustomPrompt := customPrompt.Enabled
	customPromptText := customPrompt.Prompt

	customPromptDesc := "Add extra instructions for the LLM"
	if customPrompt.Enabled && customPrompt.Prompt != "" {
		preview := customPrompt.Prompt
		if len(preview) > 40 {
			preview = preview[:40] + "..."
		}
		customPromptDesc = fmt.Sprintf("Currently: \"%s\"", preview)
	} else {
		customPromptDesc = "Currently: none. " + customPromptDesc
	}

	customForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add custom prompt?").
				Description(customPromptDesc).
				Value(&enableCustomPrompt),
		),
	).WithTheme(getTheme())

	if err := customForm.Run(); err != nil {
		return configuredProviders, err
	}

	if enableCustomPrompt {
		promptForm := huh.NewForm(
			huh.NewGroup(
				huh.NewText().
					Title("Custom Prompt").
					Description("Additional instructions (e.g., 'Format as bullet points')").
					Value(&customPromptText).
					CharLimit(500),
			),
		).WithTheme(getTheme())

		if err := promptForm.Run(); err != nil {
			return configuredProviders, err
		}
		cfg.LLM.CustomPrompt.Enabled = true
		cfg.LLM.CustomPrompt.Prompt = customPromptText
	} else {
		cfg.LLM.CustomPrompt.Enabled = false
	}

	cfg.LLM.Enabled = true
	return configuredProviders, nil
}

// getUnconfiguredLLMOptions returns options for LLM providers not yet configured
func getUnconfiguredLLMOptions(configuredProviders []string) []huh.Option[string] {
	configured := make(map[string]bool)
	for _, p := range configuredProviders {
		configured[p] = true
	}

	var options []huh.Option[string]
	if !configured["openai"] {
		options = append(options, huh.NewOption("OpenAI GPT (not configured)", "openai"))
	}
	if !configured["groq"] {
		options = append(options, huh.NewOption("Groq Llama (not configured)", "groq"))
	}
	return options
}

func getLLMModelOptions(provider string) []huh.Option[string] {
	switch provider {
	case "openai":
		return []huh.Option[string]{
			huh.NewOption("gpt-4o-mini (recommended)", "gpt-4o-mini"),
			huh.NewOption("gpt-4o", "gpt-4o"),
			huh.NewOption("gpt-4-turbo", "gpt-4-turbo"),
			huh.NewOption("gpt-3.5-turbo", "gpt-3.5-turbo"),
		}
	case "groq":
		return []huh.Option[string]{
			huh.NewOption("llama-3.3-70b-versatile (recommended)", "llama-3.3-70b-versatile"),
			huh.NewOption("llama-3.1-8b-instant (faster)", "llama-3.1-8b-instant"),
			huh.NewOption("mixtral-8x7b-32768", "mixtral-8x7b-32768"),
		}
	default:
		return []huh.Option[string]{}
	}
}

// selectPostProcessingOptions shows a multi-select for LLM post-processing toggles
func selectPostProcessingOptions(current config.LLMPostProcessingConfig) (config.LLMPostProcessingConfig, error) {
	type ppOption string
	const (
		optRemoveStutters    ppOption = "stutters"
		optAddPunctuation    ppOption = "punctuation"
		optFixGrammar        ppOption = "grammar"
		optRemoveFillerWords ppOption = "fillers"
	)

	options := []huh.Option[ppOption]{
		huh.NewOption("Remove stutters (repeated words)", optRemoveStutters),
		huh.NewOption("Add punctuation", optAddPunctuation),
		huh.NewOption("Fix grammar", optFixGrammar),
		huh.NewOption("Remove filler words (um, uh, like)", optRemoveFillerWords),
	}

	var selected []ppOption
	if current.RemoveStutters {
		selected = append(selected, optRemoveStutters)
	}
	if current.AddPunctuation {
		selected = append(selected, optAddPunctuation)
	}
	if current.FixGrammar {
		selected = append(selected, optFixGrammar)
	}
	if current.RemoveFillerWords {
		selected = append(selected, optRemoveFillerWords)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[ppOption]().
				Title("Post-Processing Options").
				Description("Select which improvements to apply").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return current, err
	}

	result := config.LLMPostProcessingConfig{}
	for _, opt := range selected {
		switch opt {
		case optRemoveStutters:
			result.RemoveStutters = true
		case optAddPunctuation:
			result.AddPunctuation = true
		case optFixGrammar:
			result.FixGrammar = true
		case optRemoveFillerWords:
			result.RemoveFillerWords = true
		}
	}

	return result, nil
}
