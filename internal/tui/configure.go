package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// ConfigureResult holds the configuration result from the TUI
type ConfigureResult struct {
	Config    *config.Config
	Cancelled bool
}

// Run starts the TUI configuration wizard
func Run(existingConfig *config.Config) (*ConfigureResult, error) {
	return runFreshInstall(existingConfig)
}

// runFreshInstall runs the full configuration wizard for fresh installs
func runFreshInstall(cfg *config.Config) (*ConfigureResult, error) {
	// Welcome screen
	fmt.Println(Logo())
	fmt.Println()
	fmt.Println(StyleMuted.Render("Voice-powered typing for Wayland/Hyprland"))
	fmt.Println()

	// Step 1: Provider selection
	selectedProviders, err := selectProviders()
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	if len(selectedProviders) == 0 {
		return &ConfigureResult{Cancelled: true}, fmt.Errorf("no providers selected")
	}

	// Initialize providers map
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}

	// Step 2: API keys for selected providers
	for _, providerName := range selectedProviders {
		apiKey, err := inputAPIKey(providerName)
		if err != nil {
			return &ConfigureResult{Cancelled: true}, nil
		}
		cfg.Providers[providerName] = config.ProviderConfig{APIKey: apiKey}
	}

	// Step 3: Transcription configuration
	transcriptionProvider, transcriptionModel, language, err := configureTranscription(selectedProviders, cfg)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	cfg.Transcription.Provider = transcriptionProvider
	cfg.Transcription.Model = transcriptionModel
	cfg.Transcription.Language = language

	// Step 4: LLM configuration
	llmEnabled, llmProvider, llmModel, postProcessing, customPrompt, err := configureLLM(selectedProviders, cfg)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	cfg.LLM.Enabled = llmEnabled
	cfg.LLM.Provider = llmProvider
	cfg.LLM.Model = llmModel
	cfg.LLM.PostProcessing = postProcessing
	cfg.LLM.CustomPrompt = customPrompt

	// Step 5: Keywords
	keywords, err := inputKeywords(cfg.Keywords)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	cfg.Keywords = keywords

	// Step 6: Injection backends
	backends, err := selectBackends(cfg.Injection.Backends)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	cfg.Injection.Backends = backends

	// Step 7: Notifications
	notificationsEnabled, err := configureNotifications(cfg.Notifications.Enabled)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	cfg.Notifications.Enabled = notificationsEnabled

	// Step 8: Summary and confirm
	confirmed, err := showSummary(cfg)
	if err != nil || !confirmed {
		return &ConfigureResult{Cancelled: true}, nil
	}

	return &ConfigureResult{Config: cfg, Cancelled: false}, nil
}

func selectProviders() ([]string, error) {
	allProviders := []string{"openai", "groq", "mistral", "elevenlabs"}

	options := []huh.Option[string]{
		huh.NewOption("OpenAI - Whisper transcription + GPT for LLM", "openai"),
		huh.NewOption("Groq - Fast Whisper transcription + Llama for LLM", "groq"),
		huh.NewOption("Mistral - Voxtral transcription (European languages)", "mistral"),
		huh.NewOption("ElevenLabs - Scribe transcription (99 languages)", "elevenlabs"),
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which providers do you want to configure?").
				Description("Select all providers you have API keys for").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Validate selected providers exist
	valid := make([]string, 0)
	for _, s := range selected {
		for _, p := range allProviders {
			if s == p {
				valid = append(valid, s)
				break
			}
		}
	}

	return valid, nil
}

func inputAPIKey(providerName string) (string, error) {
	p := provider.GetProvider(providerName)
	displayName := strings.Title(providerName)
	if p != nil {
		displayName = strings.Title(p.Name())
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

func configureTranscription(configuredProviders []string, cfg *config.Config) (string, string, string, error) {
	// Filter to only transcription-capable configured providers
	var transcriptionOptions []huh.Option[string]
	for _, name := range configuredProviders {
		p := provider.GetProvider(name)
		if p != nil && p.SupportsTranscription() {
			// Map provider name to transcription provider name
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

	if len(transcriptionOptions) == 0 {
		return "", "", "", fmt.Errorf("no transcription-capable providers configured")
	}

	var selectedProvider string
	if cfg.Transcription.Provider != "" {
		selectedProvider = cfg.Transcription.Provider
	} else if len(transcriptionOptions) > 0 {
		selectedProvider = transcriptionOptions[0].Value
	}

	providerForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Transcription Provider").
				Description("Choose which service to use for speech-to-text").
				Options(transcriptionOptions...).
				Value(&selectedProvider),
		),
	).WithTheme(getTheme())

	if err := providerForm.Run(); err != nil {
		return "", "", "", err
	}

	// Get model options for selected provider
	modelOptions := getTranscriptionModelOptions(selectedProvider)
	var selectedModel string
	if cfg.Transcription.Model != "" {
		selectedModel = cfg.Transcription.Model
	} else if len(modelOptions) > 0 {
		selectedModel = modelOptions[0].Value
	}

	var language string
	if cfg.Transcription.Language != "" {
		language = cfg.Transcription.Language
	}

	modelForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Transcription Model").
				Options(modelOptions...).
				Value(&selectedModel),
			huh.NewInput().
				Title("Language").
				Description("ISO-639-1 code (e.g., 'en', 'es', 'fr') or empty for auto-detect").
				Placeholder("auto-detect").
				Value(&language),
		),
	).WithTheme(getTheme())

	if err := modelForm.Run(); err != nil {
		return "", "", "", err
	}

	return selectedProvider, selectedModel, language, nil
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

func configureLLM(configuredProviders []string, cfg *config.Config) (bool, string, string, config.LLMPostProcessingConfig, config.LLMCustomPromptConfig, error) {
	// Filter to only LLM-capable configured providers
	var llmProviders []string
	for _, name := range configuredProviders {
		p := provider.GetProvider(name)
		if p != nil && p.SupportsLLM() {
			llmProviders = append(llmProviders, name)
		}
	}

	// Default values
	postProcessing := config.LLMPostProcessingConfig{
		RemoveStutters:    true,
		AddPunctuation:    true,
		FixGrammar:        true,
		RemoveFillerWords: true,
	}
	customPrompt := config.LLMCustomPromptConfig{
		Enabled: false,
		Prompt:  "",
	}

	// If no LLM providers configured, skip LLM config
	if len(llmProviders) == 0 {
		return false, "", "", postProcessing, customPrompt, nil
	}

	// Ask if user wants LLM post-processing
	var enableLLM bool = true
	enableForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable LLM Post-Processing? (Recommended)").
				Description("LLM improves transcription by fixing grammar, removing stutters, and cleaning up text").
				Affirmative("Yes (Recommended)").
				Negative("No").
				Value(&enableLLM),
		),
	).WithTheme(getTheme())

	if err := enableForm.Run(); err != nil {
		return false, "", "", postProcessing, customPrompt, err
	}

	if !enableLLM {
		return false, "", "", postProcessing, customPrompt, nil
	}

	// LLM provider selection
	var llmOptions []huh.Option[string]
	for _, name := range llmProviders {
		p := provider.GetProvider(name)
		if p != nil {
			switch name {
			case "openai":
				llmOptions = append(llmOptions, huh.NewOption("OpenAI GPT", "openai"))
			case "groq":
				llmOptions = append(llmOptions, huh.NewOption("Groq Llama (fast)", "groq"))
			}
		}
	}

	var selectedProvider string
	if cfg.LLM.Provider != "" {
		selectedProvider = cfg.LLM.Provider
	} else if len(llmOptions) > 0 {
		selectedProvider = llmOptions[0].Value
	}

	providerForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("LLM Provider").
				Description("Choose which service to use for text post-processing").
				Options(llmOptions...).
				Value(&selectedProvider),
		),
	).WithTheme(getTheme())

	if err := providerForm.Run(); err != nil {
		return false, "", "", postProcessing, customPrompt, err
	}

	// Model selection
	modelOptions := getLLMModelOptions(selectedProvider)
	var selectedModel string
	if cfg.LLM.Model != "" {
		selectedModel = cfg.LLM.Model
	} else if len(modelOptions) > 0 {
		selectedModel = modelOptions[0].Value
	}

	modelForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("LLM Model").
				Options(modelOptions...).
				Value(&selectedModel),
		),
	).WithTheme(getTheme())

	if err := modelForm.Run(); err != nil {
		return false, "", "", postProcessing, customPrompt, err
	}

	// Post-processing options
	if cfg.LLM.PostProcessing.RemoveStutters || cfg.LLM.PostProcessing.AddPunctuation ||
		cfg.LLM.PostProcessing.FixGrammar || cfg.LLM.PostProcessing.RemoveFillerWords {
		postProcessing = cfg.LLM.PostProcessing
	}

	ppForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Remove stutters").
				Description("Remove repeated words like 'I I I think'").
				Value(&postProcessing.RemoveStutters),
			huh.NewConfirm().
				Title("Add punctuation").
				Description("Add proper punctuation to text").
				Value(&postProcessing.AddPunctuation),
			huh.NewConfirm().
				Title("Fix grammar").
				Description("Correct grammatical errors").
				Value(&postProcessing.FixGrammar),
			huh.NewConfirm().
				Title("Remove filler words").
				Description("Remove 'um', 'uh', 'like', etc.").
				Value(&postProcessing.RemoveFillerWords),
		),
	).WithTheme(getTheme())

	if err := ppForm.Run(); err != nil {
		return false, "", "", postProcessing, customPrompt, err
	}

	// Custom prompt
	var enableCustomPrompt bool
	var customPromptText string
	if cfg.LLM.CustomPrompt.Enabled {
		enableCustomPrompt = true
		customPromptText = cfg.LLM.CustomPrompt.Prompt
	}

	customForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add custom prompt?").
				Description("Add extra instructions for the LLM").
				Value(&enableCustomPrompt),
		),
	).WithTheme(getTheme())

	if err := customForm.Run(); err != nil {
		return false, "", "", postProcessing, customPrompt, err
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
			return false, "", "", postProcessing, customPrompt, err
		}
		customPrompt.Enabled = true
		customPrompt.Prompt = customPromptText
	}

	return true, selectedProvider, selectedModel, postProcessing, customPrompt, nil
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

func inputKeywords(existingKeywords []string) ([]string, error) {
	var keywordsInput string
	if len(existingKeywords) > 0 {
		keywordsInput = strings.Join(existingKeywords, ", ")
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Keywords").
				Description("Comma-separated words to help with spelling (names, technical terms, etc.)").
				Placeholder("e.g., Kubernetes, PostgreSQL, John Smith").
				Value(&keywordsInput),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Parse keywords
	if keywordsInput == "" {
		return nil, nil
	}

	parts := strings.Split(keywordsInput, ",")
	keywords := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			keywords = append(keywords, p)
		}
	}

	return keywords, nil
}

func selectBackends(existingBackends []string) ([]string, error) {
	options := []huh.Option[string]{
		huh.NewOption("ydotool - Best for Chromium/Electron (needs ydotoold)", "ydotool"),
		huh.NewOption("wtype - Native Wayland typing", "wtype"),
		huh.NewOption("clipboard - Copy to clipboard only", "clipboard"),
	}

	var selected []string
	if len(existingBackends) > 0 {
		selected = existingBackends
	} else {
		selected = []string{"ydotool", "wtype", "clipboard"}
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Text Injection Backends").
				Description("Backends are tried in order until one succeeds (fallback chain)").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return nil, err
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("at least one backend required")
	}

	return selected, nil
}

func configureNotifications(existingEnabled bool) (bool, error) {
	enabled := existingEnabled

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable desktop notifications?").
				Description("Show notifications for recording status changes").
				Value(&enabled),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return false, err
	}

	return enabled, nil
}

func showSummary(cfg *config.Config) (bool, error) {
	fmt.Println()
	fmt.Println(StyleHeader.Render("Configuration Summary"))
	fmt.Println()

	// Providers
	var providers []string
	for name := range cfg.Providers {
		providers = append(providers, name)
	}
	fmt.Printf("  %s %s\n", StyleLabel.Render("Providers:"), strings.Join(providers, ", "))

	// Transcription
	fmt.Printf("  %s %s (%s)\n", StyleLabel.Render("Transcription:"), cfg.Transcription.Provider, cfg.Transcription.Model)
	if cfg.Transcription.Language != "" {
		fmt.Printf("  %s %s\n", StyleLabel.Render("Language:"), cfg.Transcription.Language)
	}

	// LLM
	if cfg.LLM.Enabled {
		fmt.Printf("  %s %s (%s)\n", StyleLabel.Render("LLM:"), cfg.LLM.Provider, cfg.LLM.Model)
		var ppOpts []string
		if cfg.LLM.PostProcessing.RemoveStutters {
			ppOpts = append(ppOpts, "remove stutters")
		}
		if cfg.LLM.PostProcessing.AddPunctuation {
			ppOpts = append(ppOpts, "add punctuation")
		}
		if cfg.LLM.PostProcessing.FixGrammar {
			ppOpts = append(ppOpts, "fix grammar")
		}
		if cfg.LLM.PostProcessing.RemoveFillerWords {
			ppOpts = append(ppOpts, "remove fillers")
		}
		if len(ppOpts) > 0 {
			fmt.Printf("  %s %s\n", StyleLabel.Render("Post-processing:"), strings.Join(ppOpts, ", "))
		}
	} else {
		fmt.Printf("  %s disabled\n", StyleLabel.Render("LLM:"))
	}

	// Keywords
	if len(cfg.Keywords) > 0 {
		fmt.Printf("  %s %s\n", StyleLabel.Render("Keywords:"), strings.Join(cfg.Keywords, ", "))
	}

	// Backends
	fmt.Printf("  %s %s\n", StyleLabel.Render("Backends:"), strings.Join(cfg.Injection.Backends, " -> "))

	// Notifications
	if cfg.Notifications.Enabled {
		fmt.Printf("  %s enabled\n", StyleLabel.Render("Notifications:"))
	} else {
		fmt.Printf("  %s disabled\n", StyleLabel.Render("Notifications:"))
	}

	fmt.Println()

	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Save this configuration?").
				Affirmative("Save").
				Negative("Cancel").
				Value(&confirmed),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return false, err
	}

	return confirmed, nil
}

func getTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Primary colors
	t.Focused.Title = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	t.Focused.Description = lipgloss.NewStyle().Foreground(ColorMuted)
	t.Focused.Base = lipgloss.NewStyle().BorderForeground(ColorPrimary)
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(ColorSecondary)
	t.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(ColorText)

	// Blurred (unfocused)
	t.Blurred.Title = lipgloss.NewStyle().Foreground(ColorMuted)
	t.Blurred.Description = lipgloss.NewStyle().Foreground(ColorSubtle)

	return t
}
