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

// ConfigSection represents a configuration section
type ConfigSection string

const (
	SectionProviders     ConfigSection = "providers"
	SectionTranscription ConfigSection = "transcription"
	SectionLLM           ConfigSection = "llm"
	SectionKeywords      ConfigSection = "keywords"
	SectionInjection     ConfigSection = "injection"
	SectionNotifications ConfigSection = "notifications"
	SectionFullSetup     ConfigSection = "full_setup"
)

// Run starts the TUI configuration wizard
func Run(existingConfig *config.Config) (*ConfigureResult, error) {
	// Detect if config has user changes (providers configured)
	if existingConfig != nil && hasUserChanges(existingConfig) {
		return runEditExisting(existingConfig)
	}
	return runFreshInstall(existingConfig)
}

// hasUserChanges detects if config has user modifications
func hasUserChanges(cfg *config.Config) bool {
	// If providers map has entries, user has configured something
	if len(cfg.Providers) > 0 {
		return true
	}
	// If legacy api_key is set, user has configured something
	if cfg.Transcription.APIKey != "" {
		return true
	}
	return false
}

// runEditExisting runs the section-based edit flow for existing configs
func runEditExisting(cfg *config.Config) (*ConfigureResult, error) {
	fmt.Println(Logo())
	fmt.Println()
	fmt.Println(StyleMuted.Render("Configuration detected. Select sections to edit."))
	fmt.Println()

	// Section picker
	sections, err := selectSections()
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	if len(sections) == 0 {
		return &ConfigureResult{Cancelled: true}, nil
	}

	// Check if full setup requested
	for _, s := range sections {
		if s == SectionFullSetup {
			return runFreshInstall(cfg)
		}
	}

	// Track which providers are configured (for smart detection)
	configuredProviders := getConfiguredProviders(cfg)

	// Process each selected section
	for _, section := range sections {
		switch section {
		case SectionProviders:
			if err := editProviders(cfg); err != nil {
				return &ConfigureResult{Cancelled: true}, nil
			}
			configuredProviders = getConfiguredProviders(cfg)

		case SectionTranscription:
			var err error
			configuredProviders, err = editTranscription(cfg, configuredProviders)
			if err != nil {
				return &ConfigureResult{Cancelled: true}, nil
			}

		case SectionLLM:
			var err error
			configuredProviders, err = editLLM(cfg, configuredProviders)
			if err != nil {
				return &ConfigureResult{Cancelled: true}, nil
			}

		case SectionKeywords:
			keywords, err := inputKeywords(cfg.Keywords)
			if err != nil {
				return &ConfigureResult{Cancelled: true}, nil
			}
			cfg.Keywords = keywords

		case SectionInjection:
			backends, err := selectBackends(cfg.Injection.Backends)
			if err != nil {
				return &ConfigureResult{Cancelled: true}, nil
			}
			cfg.Injection.Backends = backends

		case SectionNotifications:
			enabled, err := configureNotifications(cfg.Notifications.Enabled)
			if err != nil {
				return &ConfigureResult{Cancelled: true}, nil
			}
			cfg.Notifications.Enabled = enabled
		}
	}

	// Summary and confirm
	confirmed, err := showSummary(cfg)
	if err != nil || !confirmed {
		return &ConfigureResult{Cancelled: true}, nil
	}

	return &ConfigureResult{Config: cfg, Cancelled: false}, nil
}

func selectSections() ([]ConfigSection, error) {
	options := []huh.Option[ConfigSection]{
		huh.NewOption("Providers - API keys", SectionProviders),
		huh.NewOption("Transcription - speech-to-text settings", SectionTranscription),
		huh.NewOption("LLM - post-processing settings", SectionLLM),
		huh.NewOption("Keywords - spelling hints", SectionKeywords),
		huh.NewOption("Injection - text input backends", SectionInjection),
		huh.NewOption("Notifications - desktop alerts", SectionNotifications),
		huh.NewOption("Full Setup - reconfigure everything", SectionFullSetup),
	}

	var selected []ConfigSection
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[ConfigSection]().
				Title("What do you want to configure?").
				Description("Select one or more sections to edit").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return nil, err
	}

	return selected, nil
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

// editProviders handles the providers section edit
func editProviders(cfg *config.Config) error {
	// Show current providers with option to add/edit
	allProviders := []string{"openai", "groq", "mistral", "elevenlabs"}

	var options []huh.Option[string]
	for _, name := range allProviders {
		label := strings.Title(name)
		if _, exists := cfg.Providers[name]; exists && cfg.Providers[name].APIKey != "" {
			label += " (configured)"
		}
		switch name {
		case "openai":
			options = append(options, huh.NewOption(label+" - Whisper + GPT", name))
		case "groq":
			options = append(options, huh.NewOption(label+" - Whisper + Llama", name))
		case "mistral":
			options = append(options, huh.NewOption(label+" - Voxtral", name))
		case "elevenlabs":
			options = append(options, huh.NewOption(label+" - Scribe", name))
		}
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Configure API keys for:").
				Description("Select providers to add or update API keys").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return err
	}

	// Input API keys for selected providers
	for _, providerName := range selected {
		apiKey, err := inputAPIKey(providerName)
		if err != nil {
			return err
		}
		if cfg.Providers == nil {
			cfg.Providers = make(map[string]config.ProviderConfig)
		}
		cfg.Providers[providerName] = config.ProviderConfig{APIKey: apiKey}
	}

	return nil
}

// editTranscription handles the transcription section edit with smart provider detection
func editTranscription(cfg *config.Config, configuredProviders []string) ([]string, error) {
	// Build transcription options from configured providers
	var transcriptionOptions []huh.Option[string]
	for _, name := range configuredProviders {
		p := provider.GetProvider(name)
		if p != nil && p.SupportsTranscription() {
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

	// Add options for unconfigured providers (will prompt for key)
	unconfiguredOptions := getUnconfiguredTranscriptionOptions(configuredProviders)
	if len(unconfiguredOptions) > 0 {
		transcriptionOptions = append(transcriptionOptions, unconfiguredOptions...)
	}

	if len(transcriptionOptions) == 0 {
		return configuredProviders, fmt.Errorf("no transcription providers available")
	}

	// Set default to current provider or first option
	selectedProvider := cfg.Transcription.Provider
	if selectedProvider == "" && len(transcriptionOptions) > 0 {
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
		return configuredProviders, err
	}

	// Smart detection: if provider not configured, prompt for API key
	configuredProviders = ensureProviderConfigured(cfg, selectedProvider, configuredProviders)

	cfg.Transcription.Provider = selectedProvider

	// Model selection
	modelOptions := getTranscriptionModelOptions(selectedProvider)
	selectedModel := cfg.Transcription.Model
	if selectedModel == "" && len(modelOptions) > 0 {
		selectedModel = modelOptions[0].Value
	}

	language := cfg.Transcription.Language

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
		return configuredProviders, err
	}

	cfg.Transcription.Model = selectedModel
	cfg.Transcription.Language = language

	return configuredProviders, nil
}

// editLLM handles the LLM section edit with smart provider detection
func editLLM(cfg *config.Config, configuredProviders []string) ([]string, error) {
	// Check if any LLM-capable providers are configured
	var llmProviders []string
	for _, name := range configuredProviders {
		p := provider.GetProvider(name)
		if p != nil && p.SupportsLLM() {
			llmProviders = append(llmProviders, name)
		}
	}

	// Default post-processing
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

	// Ask if user wants LLM
	enableLLM := cfg.LLM.Enabled
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
		return configuredProviders, err
	}

	if !enableLLM {
		cfg.LLM.Enabled = false
		return configuredProviders, nil
	}

	// Build LLM provider options
	var llmOptions []huh.Option[string]
	for _, name := range llmProviders {
		switch name {
		case "openai":
			llmOptions = append(llmOptions, huh.NewOption("OpenAI GPT", "openai"))
		case "groq":
			llmOptions = append(llmOptions, huh.NewOption("Groq Llama (fast)", "groq"))
		}
	}

	// Add unconfigured LLM providers
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
		return configuredProviders, err
	}

	// Smart detection: if provider not configured, prompt for API key
	configuredProviders = ensureProviderConfigured(cfg, selectedProvider, configuredProviders)

	cfg.LLM.Provider = selectedProvider

	// Model selection
	modelOptions := getLLMModelOptions(selectedProvider)
	selectedModel := cfg.LLM.Model
	if selectedModel == "" && len(modelOptions) > 0 {
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
		return configuredProviders, err
	}

	cfg.LLM.Model = selectedModel

	// Post-processing options
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
		return configuredProviders, err
	}

	cfg.LLM.PostProcessing = postProcessing

	// Custom prompt
	enableCustomPrompt := customPrompt.Enabled
	customPromptText := customPrompt.Prompt

	customForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add custom prompt?").
				Description("Add extra instructions for the LLM").
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

// getUnconfiguredTranscriptionOptions returns options for providers not yet configured
func getUnconfiguredTranscriptionOptions(configuredProviders []string) []huh.Option[string] {
	configured := make(map[string]bool)
	for _, p := range configuredProviders {
		configured[p] = true
	}

	var options []huh.Option[string]
	if !configured["openai"] {
		options = append(options, huh.NewOption("OpenAI Whisper (needs API key)", "openai"))
	}
	if !configured["groq"] {
		options = append(options,
			huh.NewOption("Groq Whisper transcription (needs API key)", "groq-transcription"),
			huh.NewOption("Groq Whisper translation (needs API key)", "groq-translation"))
	}
	if !configured["mistral"] {
		options = append(options, huh.NewOption("Mistral Voxtral (needs API key)", "mistral-transcription"))
	}
	if !configured["elevenlabs"] {
		options = append(options, huh.NewOption("ElevenLabs Scribe (needs API key)", "elevenlabs"))
	}
	return options
}

// getUnconfiguredLLMOptions returns options for LLM providers not yet configured
func getUnconfiguredLLMOptions(configuredProviders []string) []huh.Option[string] {
	configured := make(map[string]bool)
	for _, p := range configuredProviders {
		configured[p] = true
	}

	var options []huh.Option[string]
	if !configured["openai"] {
		options = append(options, huh.NewOption("OpenAI GPT (needs API key)", "openai"))
	}
	if !configured["groq"] {
		options = append(options, huh.NewOption("Groq Llama (needs API key)", "groq"))
	}
	return options
}

// ensureProviderConfigured prompts for API key if provider not configured
func ensureProviderConfigured(cfg *config.Config, selectedProvider string, configuredProviders []string) []string {
	// Map transcription provider to actual provider name
	providerName := selectedProvider
	switch selectedProvider {
	case "groq-transcription", "groq-translation":
		providerName = "groq"
	case "mistral-transcription":
		providerName = "mistral"
	}

	// Check if already configured
	for _, p := range configuredProviders {
		if p == providerName {
			return configuredProviders
		}
	}

	// Not configured - prompt for API key
	fmt.Println()
	fmt.Println(StyleMuted.Render(fmt.Sprintf("%s not configured. Please enter API key.", strings.Title(providerName))))
	apiKey, err := inputAPIKey(providerName)
	if err != nil {
		return configuredProviders
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}
	cfg.Providers[providerName] = config.ProviderConfig{APIKey: apiKey}

	return append(configuredProviders, providerName)
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
