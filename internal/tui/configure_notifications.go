package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/notify"
)

// editNotifications handles the notifications section edit with type and custom messages
func editNotifications(cfg *config.Config) error {
	enabled := cfg.Notifications.Enabled

	desc := "Show notifications for recording status changes"
	if cfg.Notifications.Enabled {
		desc = fmt.Sprintf("Currently: enabled (%s). %s", cfg.Notifications.Type, desc)
	} else {
		desc = "Currently: disabled. " + desc
	}

	enableForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable desktop notifications?").
				Description(desc).
				Value(&enabled),
		),
	).WithTheme(getTheme())

	if err := enableForm.Run(); err != nil {
		return err
	}

	cfg.Notifications.Enabled = enabled

	if !enabled {
		return nil
	}

	notifType := cfg.Notifications.Type
	if notifType == "" {
		notifType = "desktop"
	}

	typeOptions := []huh.Option[string]{
		huh.NewOption("Desktop notifications (notify-send)", "desktop"),
		huh.NewOption("Log to console only", "log"),
		huh.NewOption("None (silent)", "none"),
	}

	typeForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Notification Type").
				Description("How should notifications be displayed?").
				Options(typeOptions...).
				Value(&notifType),
		),
	).WithTheme(getTheme())

	if err := typeForm.Run(); err != nil {
		return err
	}

	cfg.Notifications.Type = notifType

	var configureMessages bool
	msgForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Configure custom notification messages?").
				Description("Customize the text shown in notifications").
				Affirmative("Yes").
				Negative("No, use defaults").
				Value(&configureMessages),
		),
	).WithTheme(getTheme())

	if err := msgForm.Run(); err != nil {
		return err
	}

	if configureMessages {
		if err := editNotificationMessages(cfg); err != nil {
			return err
		}
	}

	return nil
}

// editNotificationMessages allows editing individual notification messages
func editNotificationMessages(cfg *config.Config) error {
	for {
		var options []huh.Option[string]
		for _, def := range notify.MessageDefs {
			currentBody := def.DefaultBody
			switch def.ConfigKey {
			case "recording_started":
				if cfg.Notifications.Messages.RecordingStarted.Body != "" {
					currentBody = cfg.Notifications.Messages.RecordingStarted.Body
				}
			case "transcribing":
				if cfg.Notifications.Messages.Transcribing.Body != "" {
					currentBody = cfg.Notifications.Messages.Transcribing.Body
				}
			case "llm_processing":
				if cfg.Notifications.Messages.LLMProcessing.Body != "" {
					currentBody = cfg.Notifications.Messages.LLMProcessing.Body
				}
			case "config_reloaded":
				if cfg.Notifications.Messages.ConfigReloaded.Body != "" {
					currentBody = cfg.Notifications.Messages.ConfigReloaded.Body
				}
			case "operation_cancelled":
				if cfg.Notifications.Messages.OperationCancelled.Body != "" {
					currentBody = cfg.Notifications.Messages.OperationCancelled.Body
				}
			case "recording_aborted":
				if cfg.Notifications.Messages.RecordingAborted.Body != "" {
					currentBody = cfg.Notifications.Messages.RecordingAborted.Body
				}
			case "injection_aborted":
				if cfg.Notifications.Messages.InjectionAborted.Body != "" {
					currentBody = cfg.Notifications.Messages.InjectionAborted.Body
				}
			}

			displayBody := currentBody
			if len(displayBody) > 30 {
				displayBody = displayBody[:30] + "..."
			}

			label := fmt.Sprintf("%s: \"%s\"", def.ConfigKey, displayBody)
			options = append(options, huh.NewOption(label, def.ConfigKey))
		}
		options = append(options, huh.NewOption("Back", "back"))

		var selected string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Notification Messages").
					Description("Select a message to edit").
					Options(options...).
					Value(&selected),
			),
		).WithTheme(getTheme())

		if err := form.Run(); err != nil {
			return err
		}

		if selected == "back" {
			return nil
		}

		if err := editSingleMessage(cfg, selected); err != nil {
			continue
		}
	}
}

// editSingleMessage edits a single notification message
func editSingleMessage(cfg *config.Config, configKey string) error {
	var def notify.MessageDef
	for _, d := range notify.MessageDefs {
		if d.ConfigKey == configKey {
			def = d
			break
		}
	}

	var currentTitle, currentBody string
	switch configKey {
	case "recording_started":
		currentTitle = cfg.Notifications.Messages.RecordingStarted.Title
		currentBody = cfg.Notifications.Messages.RecordingStarted.Body
	case "transcribing":
		currentTitle = cfg.Notifications.Messages.Transcribing.Title
		currentBody = cfg.Notifications.Messages.Transcribing.Body
	case "llm_processing":
		currentTitle = cfg.Notifications.Messages.LLMProcessing.Title
		currentBody = cfg.Notifications.Messages.LLMProcessing.Body
	case "config_reloaded":
		currentTitle = cfg.Notifications.Messages.ConfigReloaded.Title
		currentBody = cfg.Notifications.Messages.ConfigReloaded.Body
	case "operation_cancelled":
		currentTitle = cfg.Notifications.Messages.OperationCancelled.Title
		currentBody = cfg.Notifications.Messages.OperationCancelled.Body
	case "recording_aborted":
		currentTitle = cfg.Notifications.Messages.RecordingAborted.Title
		currentBody = cfg.Notifications.Messages.RecordingAborted.Body
	case "injection_aborted":
		currentTitle = cfg.Notifications.Messages.InjectionAborted.Title
		currentBody = cfg.Notifications.Messages.InjectionAborted.Body
	}

	if currentTitle == "" {
		currentTitle = def.DefaultTitle
	}
	if currentBody == "" {
		currentBody = def.DefaultBody
	}

	title := currentTitle
	body := currentBody

	var fields []huh.Field
	if !def.IsError {
		fields = append(fields, huh.NewInput().
			Title("Title").
			Description(fmt.Sprintf("Default: %s", def.DefaultTitle)).
			Placeholder(def.DefaultTitle).
			Value(&title))
	}
	fields = append(fields, huh.NewInput().
		Title("Body").
		Description(fmt.Sprintf("Default: %s", def.DefaultBody)).
		Placeholder(def.DefaultBody).
		Value(&body))

	form := huh.NewForm(
		huh.NewGroup(fields...),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return err
	}

	msgConfig := config.MessageConfig{Title: title, Body: body}
	switch configKey {
	case "recording_started":
		cfg.Notifications.Messages.RecordingStarted = msgConfig
	case "transcribing":
		cfg.Notifications.Messages.Transcribing = msgConfig
	case "llm_processing":
		cfg.Notifications.Messages.LLMProcessing = msgConfig
	case "config_reloaded":
		cfg.Notifications.Messages.ConfigReloaded = msgConfig
	case "operation_cancelled":
		cfg.Notifications.Messages.OperationCancelled = msgConfig
	case "recording_aborted":
		cfg.Notifications.Messages.RecordingAborted = msgConfig
	case "injection_aborted":
		cfg.Notifications.Messages.InjectionAborted = msgConfig
	}

	return nil
}
