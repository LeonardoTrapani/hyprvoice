package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/muesli/termenv"
)

// ConfigureResult holds the configuration result from the TUI
type ConfigureResult struct {
	Config    *config.Config
	Cancelled bool
}

// AllProviders is the list of all supported providers
var AllProviders = []string{"openai", "groq", "mistral", "elevenlabs"}

// providerDisplayNames maps provider IDs to human-readable names
var providerDisplayNames = map[string]string{
	"openai":     "OpenAI",
	"groq":       "Groq",
	"mistral":    "Mistral",
	"elevenlabs": "ElevenLabs",
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
	SectionAdvanced      ConfigSection = "advanced"
	SectionSaveExit      ConfigSection = "save_exit"
	SectionDiscardExit   ConfigSection = "discard_exit"
)

// Run starts the TUI configuration wizard
func Run(existingConfig *config.Config) (*ConfigureResult, error) {
	if existingConfig != nil && hasUserChanges(existingConfig) {
		return runEditExisting(existingConfig)
	}
	return runFreshInstall(existingConfig)
}

// hasUserChanges detects if config has user modifications
func hasUserChanges(cfg *config.Config) bool {
	if len(cfg.Providers) > 0 {
		return true
	}
	if cfg.Transcription.APIKey != "" {
		return true
	}
	return false
}

// runEditExisting runs the menu-based edit flow for existing configs
func runEditExisting(cfg *config.Config) (*ConfigureResult, error) {
	fmt.Println(Logo())
	fmt.Println()

	configuredProviders := getConfiguredProviders(cfg)

	for {
		clearScreen()
		fmt.Println(Logo())
		fmt.Println()

		section, err := selectSection(cfg)
		if err != nil {
			return &ConfigureResult{Cancelled: true}, nil
		}

		switch section {
		case SectionSaveExit:
			confirmed, err := showSummary(cfg)
			if err != nil {
				return &ConfigureResult{Cancelled: true}, nil
			}
			if confirmed {
				return &ConfigureResult{Config: cfg, Cancelled: false}, nil
			}

		case SectionDiscardExit:
			return &ConfigureResult{Cancelled: true}, nil

		case SectionProviders:
			if err := editProviders(cfg); err != nil {
				continue
			}
			configuredProviders = getConfiguredProviders(cfg)

		case SectionTranscription:
			var err error
			configuredProviders, err = editTranscription(cfg, configuredProviders)
			if err != nil {
				continue
			}

		case SectionLLM:
			var err error
			configuredProviders, err = editLLM(cfg, configuredProviders)
			if err != nil {
				continue
			}

		case SectionKeywords:
			keywords, err := inputKeywords(cfg.Keywords)
			if err != nil {
				continue
			}
			cfg.Keywords = keywords

		case SectionInjection:
			backends, err := selectBackends(cfg.Injection.Backends)
			if err != nil {
				continue
			}
			cfg.Injection.Backends = backends

		case SectionNotifications:
			if err := editNotifications(cfg); err != nil {
				continue
			}

		case SectionAdvanced:
			if err := editAdvanced(cfg); err != nil {
				continue
			}
		}
	}
}

func selectSection(cfg *config.Config) (ConfigSection, error) {
	options := []huh.Option[ConfigSection]{
		huh.NewOption(formatProvidersLabel(cfg), SectionProviders),
		huh.NewOption(formatTranscriptionLabel(cfg), SectionTranscription),
		huh.NewOption(formatLLMLabel(cfg), SectionLLM),
		huh.NewOption(formatKeywordsLabel(cfg), SectionKeywords),
		huh.NewOption(formatInjectionLabel(cfg), SectionInjection),
		huh.NewOption(formatNotificationsLabel(cfg), SectionNotifications),
		huh.NewOption("Advanced Settings", SectionAdvanced),
		huh.NewOption("Save & Exit", SectionSaveExit),
		huh.NewOption("Discard & Exit", SectionDiscardExit),
	}

	var selected ConfigSection
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[ConfigSection]().
				Title("Configuration Menu").
				Description("↑/↓ navigate • enter select • esc cancel").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return "", err
	}

	return selected, nil
}

// clearScreen clears the terminal screen
func clearScreen() {
	output := termenv.NewOutput(os.Stdout)
	output.ClearScreen()
}

func getTheme() *huh.Theme {
	t := huh.ThemeBase()

	t.Focused.Title = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	t.Focused.Description = lipgloss.NewStyle().Foreground(ColorMuted)
	t.Focused.Base = lipgloss.NewStyle().BorderForeground(ColorPrimary)
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(ColorSecondary)
	t.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(ColorText)

	t.Blurred.Title = lipgloss.NewStyle().Foreground(ColorMuted)
	t.Blurred.Description = lipgloss.NewStyle().Foreground(ColorSubtle)

	return t
}
