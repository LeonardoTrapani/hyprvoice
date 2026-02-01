package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// getModelLanguageOptions returns language options supported by the given model
func getModelLanguageOptions(model *provider.Model, currentLang string) []huh.Option[string] {
	var options []huh.Option[string]

	// auto-detect is always first
	autoLabel := "Auto-detect (recommended)"
	if currentLang == "" {
		autoLabel += " (current)"
	}
	options = append(options, huh.NewOption(autoLabel, ""))

	if model == nil {
		return options
	}

	for _, code := range model.SupportedLanguages {
		label := code
		if code == currentLang {
			label += " (current)"
		}
		options = append(options, huh.NewOption(label, code))
	}

	return options
}
