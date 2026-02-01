package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/language"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// getLanguageOptions returns language options for the dropdown
// if currentModel is provided, languages unsupported by that model will be marked
func getLanguageOptions(currentModel *provider.Model) []huh.Option[string] {
	var options []huh.Option[string]

	// auto-detect is always first and recommended
	options = append(options, huh.NewOption("Auto-detect (Recommended)", ""))

	// add all languages
	for _, lang := range language.List() {
		label := formatLanguageLabel(lang)

		// add warning if model doesn't support this language
		if currentModel != nil && !currentModel.SupportsLanguage(lang.Code) {
			label += " (not supported by current model)"
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
