package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/language"
)

// formatProvidersLabel formats the providers menu option
func formatProvidersLabel(cfg *config.Config) string {
	return "Providers"
}

// formatLanguageMenuLabel formats the language menu option showing current setting
func formatLanguageMenuLabel(cfg *config.Config) string {
	langCode := cfg.General.Language
	if langCode == "" {
		return "Language (Auto-detect)"
	}
	lang := language.FromCode(langCode)
	return fmt.Sprintf("Language (%s)", lang.Name)
}

// formatTranscriptionLabel formats the transcription menu option
func formatTranscriptionLabel(cfg *config.Config) string {
	return "Transcription"
}

// formatLLMLabel formats the LLM menu option
func formatLLMLabel(cfg *config.Config) string {
	return "LLM"
}

// formatKeywordsLabel formats the keywords menu option
func formatKeywordsLabel(cfg *config.Config) string {
	return "Keywords"
}

// formatInjectionLabel formats the injection menu option
func formatInjectionLabel(cfg *config.Config) string {
	return "Injection"
}

// formatNotificationsLabel formats the notifications menu option
func formatNotificationsLabel(cfg *config.Config) string {
	return "Notifications"
}

func showSummary(cfg *config.Config) (bool, error) {
	fmt.Println()
	fmt.Println(StyleHeader.Render("Configuration Summary"))
	fmt.Println()

	var providers []string
	for name := range cfg.Providers {
		providers = append(providers, name)
	}
	fmt.Printf("  %s %s\n", StyleLabel.Render("Providers:"), strings.Join(providers, ", "))

	fmt.Printf("  %s %s (%s)\n", StyleLabel.Render("Transcription:"), cfg.Transcription.Provider, cfg.Transcription.Model)
	if cfg.Transcription.Language != "" {
		fmt.Printf("  %s %s\n", StyleLabel.Render("Language:"), cfg.Transcription.Language)
	}

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

	if len(cfg.Keywords) > 0 {
		fmt.Printf("  %s %s\n", StyleLabel.Render("Keywords:"), strings.Join(cfg.Keywords, ", "))
	}

	fmt.Printf("  %s %s\n", StyleLabel.Render("Backends:"), strings.Join(cfg.Injection.Backends, " -> "))

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
