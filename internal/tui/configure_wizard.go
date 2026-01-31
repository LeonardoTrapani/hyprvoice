package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/config"
)

// runFreshInstall runs the full configuration wizard for fresh installs
func runFreshInstall(cfg *config.Config) (*ConfigureResult, error) {
	fmt.Println(Logo())
	fmt.Println()
	fmt.Println(StyleMuted.Render("Voice-powered typing for Wayland/Hyprland"))
	fmt.Println()

	selectedProviders, err := selectProviders()
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	if len(selectedProviders) == 0 {
		return &ConfigureResult{Cancelled: true}, fmt.Errorf("no providers selected")
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}

	for _, providerName := range selectedProviders {
		apiKey, err := inputAPIKey(providerName)
		if err != nil {
			return &ConfigureResult{Cancelled: true}, nil
		}
		cfg.Providers[providerName] = config.ProviderConfig{APIKey: apiKey}
	}

	transcriptionProvider, transcriptionModel, language, err := configureTranscription(selectedProviders, cfg)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	cfg.Transcription.Provider = transcriptionProvider
	cfg.Transcription.Model = transcriptionModel
	cfg.Transcription.Language = language

	llmEnabled, llmProvider, llmModel, postProcessing, customPrompt, err := configureLLM(selectedProviders, cfg)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	cfg.LLM.Enabled = llmEnabled
	cfg.LLM.Provider = llmProvider
	cfg.LLM.Model = llmModel
	cfg.LLM.PostProcessing = postProcessing
	cfg.LLM.CustomPrompt = customPrompt

	keywords, err := inputKeywords(cfg.Keywords)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	cfg.Keywords = keywords

	backends, err := selectBackends(cfg.Injection.Backends)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	cfg.Injection.Backends = backends

	notificationsEnabled, err := configureNotifications(cfg.Notifications.Enabled)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	cfg.Notifications.Enabled = notificationsEnabled

	confirmed, err := showSummary(cfg)
	if err != nil || !confirmed {
		return &ConfigureResult{Cancelled: true}, nil
	}

	return &ConfigureResult{Config: cfg, Cancelled: false}, nil
}

func selectProviders() ([]string, error) {
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

	valid := make([]string, 0)
	for _, s := range selected {
		for _, p := range AllProviders {
			if s == p {
				valid = append(valid, s)
				break
			}
		}
	}

	return valid, nil
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

	desc := "Show notifications for recording status changes"
	if existingEnabled {
		desc = "Currently: enabled. " + desc
	} else {
		desc = "Currently: disabled. " + desc
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable desktop notifications?").
				Description(desc).
				Value(&enabled),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return false, err
	}

	return enabled, nil
}
