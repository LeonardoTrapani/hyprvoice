package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/language"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// editLanguage allows the user to select the global transcription language
func editLanguage(cfg *config.Config) error {
	// no model-specific warnings for global language selection
	languageOptions := getLanguageOptions(nil)

	selectedLanguage := cfg.General.Language

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Language").
				Description("Select language for transcription (applies globally)").
				Options(languageOptions...).
				Filtering(true).
				Value(&selectedLanguage),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return err
	}

	// check if current transcription model supports the selected language
	if selectedLanguage != "" && cfg.Transcription.Provider != "" && cfg.Transcription.Model != "" {
		registryName := mapConfigProviderToRegistry(cfg.Transcription.Provider)
		model, err := provider.GetModel(registryName, cfg.Transcription.Model)
		if err == nil && !model.SupportsLanguage(selectedLanguage) {
			langName := language.FromCode(selectedLanguage).Name
			if langName == "" {
				langName = selectedLanguage
			}

			fmt.Println()
			fmt.Println(StyleWarning.Render("Language-Model Compatibility Warning"))
			fmt.Printf("Your current model '%s' does not support %s.\n", model.Name, langName)
			fmt.Println()
			fmt.Println(StyleMuted.Render("You can:"))
			fmt.Println(StyleMuted.Render("  - Keep this language and change the model later"))
			fmt.Println(StyleMuted.Render("  - Use 'Auto-detect' for language"))
			fmt.Println(StyleMuted.Render("  - Choose a different language"))
			fmt.Println()

			var action string
			actionForm := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("What would you like to do?").
						Options(
							huh.NewOption("Keep this language (change model later)", "keep"),
							huh.NewOption("Use Auto-detect instead", "auto"),
							huh.NewOption("Choose a different language", "retry"),
						).
						Value(&action),
				),
			).WithTheme(getTheme())

			if err := actionForm.Run(); err != nil {
				return err
			}

			switch action {
			case "auto":
				selectedLanguage = ""
			case "retry":
				return editLanguage(cfg)
			case "keep":
				// proceed with incompatible language
			}
		}
	}

	cfg.General.Language = selectedLanguage
	return nil
}
