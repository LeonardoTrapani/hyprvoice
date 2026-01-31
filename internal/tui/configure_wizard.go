package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/config"
)

// runFreshInstall runs the guided onboarding flow for fresh installs
// Uses the same screens as the menu for consistency
func runFreshInstall(cfg *config.Config) (*ConfigureResult, error) {
	fmt.Println(Logo())
	fmt.Println()
	fmt.Println(StyleMuted.Render("Voice-powered typing for Wayland/Hyprland"))
	fmt.Println()

	// 1. Providers - same screen as menu
	if err := editProviders(cfg, true); err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}

	configuredProviders := getConfiguredProviders(cfg)
	if len(configuredProviders) == 0 {
		return &ConfigureResult{Cancelled: true}, fmt.Errorf("no providers configured")
	}

	// 2. Transcription - same screen as menu
	var err error
	configuredProviders, err = editTranscription(cfg, configuredProviders)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}

	// 3. LLM - same screen as menu
	configuredProviders, err = editLLM(cfg, configuredProviders)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}

	// 4. Keywords
	keywords, err := inputKeywords(cfg.Keywords)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	cfg.Keywords = keywords

	// 5. Injection backends
	backends, err := selectBackends(cfg.Injection.Backends)
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	cfg.Injection.Backends = backends

	// 6. Notifications - same screen as menu
	if err := editNotifications(cfg); err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}

	// 7. Advanced settings prompt
	wantAdvanced, err := askAdvancedSettings()
	if err != nil {
		return &ConfigureResult{Cancelled: true}, nil
	}
	if wantAdvanced {
		if err := editAdvanced(cfg, true); err != nil {
			return &ConfigureResult{Cancelled: true}, nil
		}
	}

	return &ConfigureResult{Config: cfg, Cancelled: false}, nil
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

func askAdvancedSettings() (bool, error) {
	var want bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Configure advanced settings?").
				Description("Recording parameters, injection timeouts, etc.").
				Value(&want),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return false, err
	}
	return want, nil
}
