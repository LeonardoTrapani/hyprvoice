package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/language"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// getModelLanguageOptions returns language options supported by the given model
func getModelLanguageOptions(model *provider.Model, currentLang string) []huh.Option[string] {
	var options []huh.Option[string]

	// auto-detect is always first
	autoLabel := "Auto-detect"
	if currentLang == "" {
		autoLabel += " (current)"
	}
	options = append(options, huh.NewOption(autoLabel, ""))

	// only show languages supported by the model
	for _, lang := range language.List() {
		if model != nil && !model.SupportsLanguage(lang.Code) {
			continue
		}

		label := formatLanguageLabel(lang)
		if lang.Code == currentLang {
			label += " (current)"
		}

		options = append(options, huh.NewOption(label, lang.Code))
	}

	return options
}

// formatLanguageLabel formats a language for display
func formatLanguageLabel(lang language.Language) string {
	if lang.Name == lang.NativeName || lang.NativeName == "" {
		return fmt.Sprintf("%s (%s)", lang.Name, lang.Code)
	}
	return fmt.Sprintf("%s - %s (%s)", lang.Name, lang.NativeName, lang.Code)
}
