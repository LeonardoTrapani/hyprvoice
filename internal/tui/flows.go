package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/deps"
	"github.com/leonardotrapani/hyprvoice/internal/models/whisper"
	"github.com/leonardotrapani/hyprvoice/internal/notify"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

const (
	menuProviders     = "providers"
	menuVoiceModel    = "voice_model"
	menuLLM           = "llm"
	menuKeywords      = "keywords"
	menuInjection     = "injection"
	menuNotifications = "notifications"
	menuAdvanced      = "advanced"
	menuSave          = "save"
	menuDiscard       = "discard"
)

func newWelcomeScreen(state *wizardState) screen {
	desc := []string{
		"Voice-powered typing for Wayland/Hyprland.",
		"Let's set up your configuration.",
	}
	s := newInfoScreen(state, "Hyprvoice Configure", desc, func() screen {
		return onboardingProvidersScreen(state)
	}, func() screen {
		state.cancelled = true
		state.result = &ConfigureResult{Cancelled: true}
		return nil
	})
	s.footer = "enter start • esc quit"
	return s
}

func onboardingProvidersScreen(state *wizardState) screen {
	return newProvidersScreen(state,
		func() screen { return newWelcomeScreen(state) },
		func() screen { return onboardingVoiceProviderScreen(state) },
		true,
	)
}

func onboardingVoiceProviderScreen(state *wizardState) screen {
	return newVoiceProviderScreen(state,
		func() screen { return onboardingProvidersScreen(state) },
		func() screen { return onboardingLLMScreen(state) },
	)
}

func onboardingLLMScreen(state *wizardState) screen {
	return newLLMEnableScreen(state,
		func() screen { return onboardingVoiceProviderScreen(state) },
		func() screen { return newKeywordsScreen(state, func() screen { return onboardingLLMScreen(state) }) },
	)
}

func newMenuScreen(state *wizardState) screen {
	items := []optionItem{
		{title: formatProvidersLabel(state.cfg), value: menuProviders},
		{title: formatVoiceModelLabel(state.cfg), value: menuVoiceModel},
		{title: formatLLMLabel(state.cfg), value: menuLLM},
		{title: formatKeywordsLabel(state.cfg), value: menuKeywords},
		{title: formatInjectionLabel(state.cfg), value: menuInjection},
		{title: formatNotificationsLabel(state.cfg), value: menuNotifications},
		{title: "Advanced Settings", value: menuAdvanced},
		{title: "Save & Exit", value: menuSave},
		{title: "Discard & Exit", value: menuDiscard},
	}

	desc := []string{"Select a section to update."}
	screen := newListScreen(state, "Configuration Menu", desc, items, func(item optionItem) screen {
		switch item.value {
		case menuProviders:
			return newProvidersScreen(state, func() screen { return newMenuScreen(state) }, func() screen { return newMenuScreen(state) }, false)
		case menuVoiceModel:
			return newVoiceProviderScreen(state, func() screen { return newMenuScreen(state) }, func() screen { return newMenuScreen(state) })
		case menuLLM:
			return newLLMEnableScreen(state, func() screen { return newMenuScreen(state) }, func() screen { return newMenuScreen(state) })
		case menuKeywords:
			return newKeywordsScreen(state, func() screen { return newMenuScreen(state) })
		case menuInjection:
			return newInjectionScreen(state, func() screen { return newMenuScreen(state) })
		case menuNotifications:
			return newNotificationsScreen(state, func() screen { return newMenuScreen(state) })
		case menuAdvanced:
			return newAdvancedMenuScreen(state, func() screen { return newMenuScreen(state) }, false)
		case menuSave:
			return newSummaryScreen(state, func() screen { return newMenuScreen(state) })
		case menuDiscard:
			state.cancelled = true
			state.result = &ConfigureResult{Cancelled: true}
			return nil
		default:
			return newMenuScreen(state)
		}
	}, func() screen {
		state.cancelled = true
		state.result = &ConfigureResult{Cancelled: true}
		return nil
	})
	screen.footer = "enter select • esc cancel • / filter"
	return screen
}

func newProvidersScreen(state *wizardState, onBack func() screen, onNext func() screen, onboarding bool) screen {
	items := make([]optionItem, 0, len(AllProviders)+1)
	for _, name := range AllProviders {
		items = append(items, optionItem{title: formatProviderOption(state.cfg, name), value: name})
	}

	exitLabel := "Done"
	if onboarding {
		exitLabel = "Next"
	}
	items = append(items, optionItem{title: exitLabel, value: "back"})

	desc := []string{
		"Add or update API keys for cloud providers.",
		"Recommended: local models maximize privacy; for cloud quality, ElevenLabs is the top pick.",
		"Tip: press / to filter.",
	}

	screen := newListScreen(state, "Provider API Keys", desc, items, func(item optionItem) screen {
		if item.value == "back" {
			if onboarding && onNext != nil {
				return onNext()
			}
			if onBack != nil {
				return onBack()
			}
			return nil
		}
		return newProviderKeyFlow(state, item.value, func() screen { return newProvidersScreen(state, onBack, onNext, onboarding) }, func() screen { return newProvidersScreen(state, onBack, onNext, onboarding) })
	}, func() screen {
		if onBack != nil {
			return onBack()
		}
		state.cancelled = true
		state.result = &ConfigureResult{Cancelled: true}
		return nil
	})
	screen.footer = "enter select • esc back • / filter"
	return screen
}

func newProviderKeyFlow(state *wizardState, providerName string, onContinue func() screen, onCancel func() screen) screen {
	displayName := getProviderDisplayName(providerName)
	if isProviderConfigured(state.cfg, providerName) {
		masked := maskAPIKey(state.cfg.Providers[providerName].APIKey)
		desc := []string{fmt.Sprintf("Current key: %s", masked)}
		return newConfirmScreen(state, fmt.Sprintf("%s API Key", displayName), desc, "Update key", "Keep current", func() screen {
			return newAPIKeyInputScreen(state, providerName, onContinue, onCancel)
		}, func() screen { return onContinue() })
	}
	return newAPIKeyInputScreen(state, providerName, onContinue, onCancel)
}

func newAPIKeyInputScreen(state *wizardState, providerName string, onContinue func() screen, onCancel func() screen) screen {
	p := provider.GetProvider(providerName)
	displayName := getProviderDisplayName(providerName)
	if p != nil {
		if name, ok := providerDisplayNames[p.Name()]; ok {
			displayName = name
		}
	}
	desc := []string{fmt.Sprintf("Enter your %s API key", displayName)}
	validate := func(s string) error {
		if s == "" {
			return fmt.Errorf("API key is required")
		}
		if p != nil && !p.ValidateAPIKey(s) {
			return fmt.Errorf("invalid API key format for %s", displayName)
		}
		return nil
	}
	return newInputScreen(state, fmt.Sprintf("%s API Key", displayName), desc, "", "", true, validate, func(value string) screen {
		if state.cfg.Providers == nil {
			state.cfg.Providers = make(map[string]config.ProviderConfig)
		}
		state.cfg.Providers[providerName] = config.ProviderConfig{APIKey: value}
		return onContinue()
	}, onCancel)
}

func newVoiceProviderScreen(state *wizardState, onBack func() screen, onNext func() screen) screen {
	options := buildVoiceProviderOptions(state.cfg)
	if len(options) == 0 {
		desc := []string{
			"No voice model providers are available.",
			"Configure a cloud provider API key or install whisper.cpp.",
		}
		s := newInfoScreen(state, "Voice Model Provider", desc, onBack, onBack)
		s.footer = "enter back • esc back"
		return s
	}

	desc := []string{
		"Choose the provider for speech-to-text.",
		"Recommended: local models maximize privacy; for cloud quality, ElevenLabs is the top pick.",
		"Tip: press / to filter.",
	}

	screen := newListScreen(state, "Voice Model Provider", desc, options, func(item optionItem) screen {
		if item.value == "whisper-cpp-disabled" {
			info := []string{
				"whisper-cli was not found in PATH.",
				"Install whisper.cpp to use local transcription:",
				"https://github.com/ggerganov/whisper.cpp",
			}
			s := newInfoScreen(state, "Whisper.cpp Not Found", info, func() screen {
				return newVoiceProviderScreen(state, onBack, onNext)
			}, func() screen { return newVoiceProviderScreen(state, onBack, onNext) })
			s.footer = "enter choose another • esc back"
			return s
		}

		selectedProvider := item.value
		providerName := selectedProvider
		switch selectedProvider {
		case "groq-transcription", "groq-translation":
			providerName = "groq"
		case "mistral-transcription":
			providerName = "mistral"
		}

		if selectedProvider != "whisper-cpp" && !isProviderConfigured(state.cfg, providerName) {
			return newProviderKeyFlow(state, providerName, func() screen {
				return newVoiceModelScreen(state, selectedProvider, onBack, onNext)
			}, func() screen { return newVoiceProviderScreen(state, onBack, onNext) })
		}

		return newVoiceModelScreen(state, selectedProvider, onBack, onNext)
	}, func() screen { return onBack() })
	screen.footer = "enter select • esc back • / filter"

	if state.cfg.Transcription.Provider != "" {
		selectListByValue(&screen.list, state.cfg.Transcription.Provider)
	}
	return screen
}

func newVoiceModelScreen(state *wizardState, providerName string, onBack func() screen, onNext func() screen) screen {
	options := getTranscriptionModelOptions(providerName)
	if len(options) == 0 {
		desc := []string{"No models available for this provider."}
		s := newInfoScreen(state, "Voice Model", desc, onBack, onBack)
		s.footer = "enter back • esc back"
		return s
	}

	items := make([]optionItem, 0, len(options))
	for _, opt := range options {
		items = append(items, optionItem{title: opt.Label, value: opt.ID})
	}

	desc := []string{
		"Pick the model for speech-to-text.",
		"Tip: press / to filter.",
	}
	screen := newListScreen(state, "Voice Model", desc, items, func(item optionItem) screen {
		if item.value == "" {
			return newVoiceModelScreen(state, providerName, onBack, onNext)
		}
		if providerName == "whisper-cpp" && !whisper.IsInstalled(item.value) {
			modelInfo := whisper.GetModel(item.value)
			if modelInfo == nil {
				state.err = fmt.Errorf("unknown model: %s", item.value)
				return nil
			}
			confirmDesc := []string{fmt.Sprintf("Download %s (%s)?", modelInfo.Name, modelInfo.Size)}
			return newConfirmScreen(state, "Download Model", confirmDesc, "Download", "Cancel", func() screen {
				return newDownloadScreen(state, "Downloading Model", []string{modelInfo.Name}, item.value, func() screen {
					return applyVoiceModelSelection(state, providerName, item.value, onBack, onNext)
				}, func() screen { return newVoiceModelScreen(state, providerName, onBack, onNext) })
			}, func() screen { return newVoiceModelScreen(state, providerName, onBack, onNext) })
		}
		return applyVoiceModelSelection(state, providerName, item.value, onBack, onNext)
	}, func() screen { return onBack() })
	screen.footer = "enter select • esc back • / filter"

	if state.cfg.Transcription.Model != "" {
		selectListByValue(&screen.list, state.cfg.Transcription.Model)
	}
	return screen
}

func applyVoiceModelSelection(state *wizardState, providerName, modelID string, onBack func() screen, onNext func() screen) screen {
	state.cfg.Transcription.Provider = providerName
	state.cfg.Transcription.Model = modelID

	registryName := mapConfigProviderToRegistry(providerName)
	model, err := provider.GetModel(registryName, modelID)
	if err != nil {
		state.err = err
		return nil
	}

	if state.cfg.Transcription.Language != "" && !model.SupportsLanguage(state.cfg.Transcription.Language) {
		state.cfg.Transcription.Language = ""
	}

	if len(model.SupportedLanguages) <= 1 {
		if len(model.SupportedLanguages) == 1 {
			state.cfg.Transcription.Language = model.SupportedLanguages[0]
		} else {
			state.cfg.Transcription.Language = ""
		}
		return applyStreamingSelection(state, model, onNext)
	}

	return newLanguageScreen(state, model, func() screen {
		return newVoiceModelScreen(state, providerName, onBack, onNext)
	}, func() screen {
		return applyStreamingSelection(state, model, onNext)
	})
}

func newLanguageScreen(state *wizardState, model *provider.Model, onBack func() screen, onNext func() screen) screen {
	items := []optionItem{{title: "Auto-detect (recommended)", value: ""}}
	for _, code := range model.SupportedLanguages {
		items = append(items, optionItem{title: code, value: code})
	}
	desc := []string{"Select the language for the voice model.", "Tip: press / to filter."}
	screen := newListScreen(state, "Language", desc, items, func(item optionItem) screen {
		state.cfg.Transcription.Language = item.value
		return onNext()
	}, func() screen { return onBack() })
	screen.footer = "enter select • esc back • / filter"

	if state.cfg.Transcription.Language != "" {
		selectListByValue(&screen.list, state.cfg.Transcription.Language)
	}
	return screen
}

func applyStreamingSelection(state *wizardState, model *provider.Model, next func() screen) screen {
	if model.SupportsBothModes() {
		desc := []string{"This model supports both batch and streaming modes."}
		return newConfirmScreen(state, "Enable Streaming Mode?", desc, "Yes, streaming", "No, batch", func() screen {
			state.cfg.Transcription.Streaming = true
			return next()
		}, func() screen {
			state.cfg.Transcription.Streaming = false
			return next()
		})
	}
	if model.SupportsStreaming {
		state.cfg.Transcription.Streaming = true
	} else {
		state.cfg.Transcription.Streaming = false
	}
	return next()
}

func newLLMEnableScreen(state *wizardState, onBack func() screen, onNext func() screen) screen {
	desc := []string{"LLM post-processing cleans up grammar, punctuation, and filler words."}
	if state.cfg.LLM.Enabled {
		desc = []string{fmt.Sprintf("Currently enabled (%s/%s).", state.cfg.LLM.Provider, state.cfg.LLM.Model), desc[0]}
	} else {
		desc = []string{"Currently disabled.", desc[0]}
	}
	return newConfirmScreen(state, "Enable LLM Post-Processing?", desc, "Yes (recommended)", "No", func() screen {
		return newLLMProviderScreen(state, onBack, onNext)
	}, func() screen {
		state.cfg.LLM.Enabled = false
		return onNext()
	})
}

func newLLMProviderScreen(state *wizardState, onBack func() screen, onNext func() screen) screen {
	options := buildLLMProviderOptions(state.cfg)
	if len(options) == 0 {
		info := []string{"No LLM providers available.", "Configure OpenAI or Groq first."}
		s := newInfoScreen(state, "LLM Provider", info, onBack, onBack)
		s.footer = "enter back • esc back"
		return s
	}

	desc := []string{"Choose a provider for text post-processing.", "Tip: press / to filter."}
	screen := newListScreen(state, "LLM Provider", desc, options, func(item optionItem) screen {
		providerName := item.value
		if !isProviderConfigured(state.cfg, providerName) {
			return newProviderKeyFlow(state, providerName, func() screen {
				return newLLMModelScreen(state, providerName, onBack, onNext)
			}, func() screen { return newLLMProviderScreen(state, onBack, onNext) })
		}
		return newLLMModelScreen(state, providerName, onBack, onNext)
	}, func() screen { return onBack() })
	screen.footer = "enter select • esc back • / filter"
	if state.cfg.LLM.Provider != "" {
		selectListByValue(&screen.list, state.cfg.LLM.Provider)
	}
	return screen
}

func newLLMModelScreen(state *wizardState, providerName string, onBack func() screen, onNext func() screen) screen {
	p := provider.GetProvider(providerName)
	if p == nil {
		state.err = fmt.Errorf("unknown provider: %s", providerName)
		return nil
	}
	models := provider.ModelsOfType(p, provider.LLM)
	items := make([]optionItem, 0, len(models))
	for _, m := range models {
		items = append(items, optionItem{title: fmt.Sprintf("%s (%s)", m.Name, m.Description), value: m.ID})
	}
	desc := []string{"Choose the LLM model.", "Tip: press / to filter."}
	screen := newListScreen(state, "LLM Model", desc, items, func(item optionItem) screen {
		state.cfg.LLM.Provider = providerName
		state.cfg.LLM.Model = item.value
		return newPostProcessingScreen(state, onBack, onNext)
	}, func() screen { return newLLMProviderScreen(state, onBack, onNext) })
	screen.footer = "enter select • esc back • / filter"
	if state.cfg.LLM.Model != "" {
		selectListByValue(&screen.list, state.cfg.LLM.Model)
	}
	return screen
}

func newPostProcessingScreen(state *wizardState, onBack func() screen, onNext func() screen) screen {
	current := state.cfg.LLM.PostProcessing
	if !current.RemoveStutters && !current.AddPunctuation && !current.FixGrammar && !current.RemoveFillerWords {
		current = config.LLMPostProcessingConfig{
			RemoveStutters:    true,
			AddPunctuation:    true,
			FixGrammar:        true,
			RemoveFillerWords: true,
		}
	}

	items := []toggleItem{
		{title: "Remove stutters (repeated words)", value: "stutters", selected: current.RemoveStutters},
		{title: "Add punctuation", value: "punctuation", selected: current.AddPunctuation},
		{title: "Fix grammar", value: "grammar", selected: current.FixGrammar},
		{title: "Remove filler words (um, uh, like)", value: "fillers", selected: current.RemoveFillerWords},
	}

	desc := []string{"Select which improvements to apply.", "Tip: press / to filter."}
	screen := newMultiSelectScreen(state, "Post-Processing Options", desc, items, true, func(items []toggleItem) screen {
		result := config.LLMPostProcessingConfig{}
		for _, item := range items {
			if !item.selected {
				continue
			}
			switch item.value {
			case "stutters":
				result.RemoveStutters = true
			case "punctuation":
				result.AddPunctuation = true
			case "grammar":
				result.FixGrammar = true
			case "fillers":
				result.RemoveFillerWords = true
			}
		}
		state.cfg.LLM.PostProcessing = result
		return newCustomPromptConfirmScreen(state, onBack, onNext)
	}, func() screen { return newLLMModelScreen(state, state.cfg.LLM.Provider, onBack, onNext) })
	screen.footer = "space toggle • enter save • esc back • / filter"
	return screen
}

func newCustomPromptConfirmScreen(state *wizardState, onBack func() screen, onNext func() screen) screen {
	desc := []string{"Add extra instructions for the LLM."}
	if state.cfg.LLM.CustomPrompt.Enabled && state.cfg.LLM.CustomPrompt.Prompt != "" {
		preview := state.cfg.LLM.CustomPrompt.Prompt
		if len(preview) > 40 {
			preview = preview[:40] + "..."
		}
		desc = append([]string{fmt.Sprintf("Current prompt: \"%s\"", preview)}, desc...)
	} else {
		desc = append([]string{"Current prompt: none."}, desc...)
	}
	return newConfirmScreen(state, "Add Custom Prompt?", desc, "Yes", "No", func() screen {
		return newInputScreen(state, "Custom Prompt", []string{"Additional instructions for the LLM."}, state.cfg.LLM.CustomPrompt.Prompt, "Format as bullet points", false, func(s string) error {
			if len(s) > 500 {
				return fmt.Errorf("prompt must be 500 characters or less")
			}
			return nil
		}, func(value string) screen {
			state.cfg.LLM.CustomPrompt.Enabled = true
			state.cfg.LLM.CustomPrompt.Prompt = value
			state.cfg.LLM.Enabled = true
			return onNext()
		}, func() screen { return newCustomPromptConfirmScreen(state, onBack, onNext) })
	}, func() screen {
		state.cfg.LLM.CustomPrompt.Enabled = false
		state.cfg.LLM.Enabled = true
		return onNext()
	})
}

func newKeywordsScreen(state *wizardState, onBack func() screen) screen {
	desc := []string{
		"Comma-separated words to keep spelling accurate (names, acronyms, terms).",
		"Used by LLM post-processing to preserve spelling and phrasing.",
	}
	initial := ""
	if len(state.cfg.Keywords) > 0 {
		initial = strings.Join(state.cfg.Keywords, ", ")
	}
	return newInputScreen(state, "Keywords", desc, initial, "e.g., Kubernetes, PostgreSQL, John Smith", false, nil, func(value string) screen {
		if strings.TrimSpace(value) == "" {
			state.cfg.Keywords = nil
		} else {
			parts := strings.Split(value, ",")
			keywords := make([]string, 0, len(parts))
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					keywords = append(keywords, part)
				}
			}
			state.cfg.Keywords = keywords
		}
		if state.onboarding {
			return newInjectionScreen(state, func() screen { return newKeywordsScreen(state, onBack) })
		}
		return onBack()
	}, onBack)
}

func newInjectionScreen(state *wizardState, onBack func() screen) screen {
	selected := state.cfg.Injection.Backends
	if len(selected) == 0 {
		selected = []string{"ydotool", "wtype", "clipboard"}
	}
	selectedSet := make(map[string]bool, len(selected))
	for _, b := range selected {
		selectedSet[b] = true
	}

	items := []toggleItem{
		{title: "ydotool - best for Chromium/Electron (needs ydotoold)", value: "ydotool", selected: selectedSet["ydotool"]},
		{title: "wtype - native Wayland typing", value: "wtype", selected: selectedSet["wtype"]},
		{title: "clipboard - copy to clipboard only", value: "clipboard", selected: selectedSet["clipboard"]},
	}
	desc := []string{"Backends are tried in order until one succeeds.", "Tip: press / to filter."}
	screen := newMultiSelectScreen(state, "Text Injection Backends", desc, items, true, func(items []toggleItem) screen {
		var backends []string
		for _, item := range items {
			if item.selected {
				backends = append(backends, item.value)
			}
		}
		state.cfg.Injection.Backends = backends
		if state.onboarding {
			return newNotificationsScreen(state, func() screen { return newInjectionScreen(state, onBack) })
		}
		return onBack()
	}, onBack)
	screen.footer = "space toggle • enter save • esc back • / filter"
	return screen
}

func newNotificationsScreen(state *wizardState, onBack func() screen) screen {
	desc := []string{"Show notifications for recording status changes."}
	if state.cfg.Notifications.Enabled {
		desc = append([]string{fmt.Sprintf("Currently enabled (%s).", state.cfg.Notifications.Type)}, desc...)
	} else {
		desc = append([]string{"Currently disabled."}, desc...)
	}
	return newConfirmScreen(state, "Enable Desktop Notifications?", desc, "Yes", "No", func() screen {
		state.cfg.Notifications.Enabled = true
		return newNotificationTypeScreen(state, onBack)
	}, func() screen {
		state.cfg.Notifications.Enabled = false
		if state.onboarding {
			return newAdvancedPromptScreen(state, onBack)
		}
		return onBack()
	})
}

func newNotificationTypeScreen(state *wizardState, onBack func() screen) screen {
	if state.cfg.Notifications.Type == "" {
		state.cfg.Notifications.Type = "desktop"
	}
	items := []optionItem{
		{title: "Desktop notifications (notify-send)", value: "desktop"},
		{title: "Log to console only", value: "log"},
		{title: "None (silent)", value: "none"},
	}
	desc := []string{"Choose how notifications should be displayed."}
	screen := newListScreen(state, "Notification Type", desc, items, func(item optionItem) screen {
		state.cfg.Notifications.Type = item.value
		return newCustomMessagesConfirmScreen(state, onBack)
	}, func() screen { return newNotificationsScreen(state, onBack) })
	screen.footer = "enter select • esc back • / filter"
	selectListByValue(&screen.list, state.cfg.Notifications.Type)
	return screen
}

func newCustomMessagesConfirmScreen(state *wizardState, onBack func() screen) screen {
	desc := []string{"Customize the text shown in notifications."}
	return newConfirmScreen(state, "Customize Notification Messages?", desc, "Yes", "No", func() screen {
		return newNotificationMessagesScreen(state, onBack)
	}, func() screen {
		if state.onboarding {
			return newAdvancedPromptScreen(state, onBack)
		}
		return onBack()
	})
}

func newNotificationMessagesScreen(state *wizardState, onBack func() screen) screen {
	items := make([]optionItem, 0, len(notify.MessageDefs)+1)
	for _, def := range notify.MessageDefs {
		_, currentBody := getNotificationMessage(state.cfg, def)
		display := currentBody
		if display == "" {
			display = def.DefaultBody
		}
		if len(display) > 40 {
			display = display[:40] + "..."
		}
		label := fmt.Sprintf("%s: \"%s\"", def.ConfigKey, display)
		items = append(items, optionItem{title: label, value: def.ConfigKey})
	}
	items = append(items, optionItem{title: "Back", value: "back"})
	desc := []string{"Select a message to edit."}
	screen := newListScreen(state, "Notification Messages", desc, items, func(item optionItem) screen {
		if item.value == "back" {
			if state.onboarding {
				return newAdvancedPromptScreen(state, onBack)
			}
			return onBack()
		}
		return newNotificationMessageEditScreen(state, item.value, func() screen { return newNotificationMessagesScreen(state, onBack) })
	}, func() screen { return onBack() })
	screen.footer = "enter select • esc back • / filter"
	return screen
}

func newNotificationMessageEditScreen(state *wizardState, configKey string, onBack func() screen) screen {
	def := findMessageDef(configKey)
	if def == nil {
		state.err = fmt.Errorf("unknown message: %s", configKey)
		return nil
	}

	currentTitle, currentBody := getNotificationMessage(state.cfg, *def)
	if currentTitle == "" {
		currentTitle = def.DefaultTitle
	}
	if currentBody == "" {
		currentBody = def.DefaultBody
	}

	fields := []formField{}
	if !def.IsError {
		fields = append(fields, makeInputField("title", "Title", fmt.Sprintf("Default: %s", def.DefaultTitle), currentTitle, def.DefaultTitle, nil))
	}
	fields = append(fields, makeInputField("body", "Body", fmt.Sprintf("Default: %s", def.DefaultBody), currentBody, def.DefaultBody, nil))

	screen := newFormScreen(state, "Edit Notification", nil, fields, func(values map[string]string) screen {
		msg := config.MessageConfig{Title: values["title"], Body: values["body"]}
		setNotificationMessage(state.cfg, configKey, msg)
		return onBack()
	}, onBack)
	screen.footer = "enter save • esc back"
	return screen
}

func newAdvancedPromptScreen(state *wizardState, onBack func() screen) screen {
	desc := []string{"Configure advanced settings like recording parameters and timeouts."}
	return newConfirmScreen(state, "Configure Advanced Settings?", desc, "Yes", "No", func() screen {
		return newAdvancedMenuScreen(state, onBack, true)
	}, func() screen {
		if state.onboarding {
			return newMenuScreen(state)
		}
		return onBack()
	})
}

func newAdvancedMenuScreen(state *wizardState, onBack func() screen, onboarding bool) screen {
	items := []optionItem{
		{title: formatAdvancedRecordingLabel(state.cfg), value: "recording"},
		{title: formatAdvancedInjectionTimeoutLabel(state.cfg), value: "timeouts"},
		{title: "Back", value: "back"},
	}
	if onboarding {
		items[len(items)-1].title = "Next"
	}
	desc := []string{"Configure low-level options."}
	screen := newListScreen(state, "Advanced Settings", desc, items, func(item optionItem) screen {
		switch item.value {
		case "recording":
			return newRecordingSettingsScreen(state, func() screen { return newAdvancedMenuScreen(state, onBack, onboarding) })
		case "timeouts":
			return newInjectionTimeoutsScreen(state, func() screen { return newAdvancedMenuScreen(state, onBack, onboarding) })
		case "back":
			if onboarding {
				return newMenuScreen(state)
			}
			return onBack()
		default:
			return newAdvancedMenuScreen(state, onBack, onboarding)
		}
	}, func() screen { return onBack() })
	screen.footer = "enter select • esc back • / filter"
	return screen
}

func newRecordingSettingsScreen(state *wizardState, onBack func() screen) screen {
	cfg := state.cfg.Recording
	fields := []formField{
		makeInputField("sample_rate", "Sample Rate (Hz)", "16000 is optimal for speech recognition.", strconv.Itoa(cfg.SampleRate), "16000", func(s string) error {
			if _, err := strconv.Atoi(s); err != nil {
				return fmt.Errorf("sample rate must be a number")
			}
			return nil
		}),
		makeInputField("channels", "Channels", "1 (mono) recommended.", strconv.Itoa(cfg.Channels), "1", func(s string) error {
			v, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("channels must be a number")
			}
			if v != 1 && v != 2 {
				return fmt.Errorf("channels must be 1 or 2")
			}
			return nil
		}),
		makeInputField("format", "Audio Format", "Use s16 for most setups.", cfg.Format, "s16", func(s string) error {
			if s != "s16" && s != "f32" {
				return fmt.Errorf("format must be s16 or f32")
			}
			return nil
		}),
		makeInputField("buffer_size", "Buffer Size (bytes)", "Larger = less CPU, more latency.", strconv.Itoa(cfg.BufferSize), "8192", func(s string) error {
			if _, err := strconv.Atoi(s); err != nil {
				return fmt.Errorf("buffer size must be a number")
			}
			return nil
		}),
		makeInputField("channel_buffer", "Channel Buffer Size", "Number of audio frames to buffer.", strconv.Itoa(cfg.ChannelBufferSize), "30", func(s string) error {
			if _, err := strconv.Atoi(s); err != nil {
				return fmt.Errorf("channel buffer size must be a number")
			}
			return nil
		}),
		makeInputField("device", "Device", "Leave empty for default microphone.", cfg.Device, "(default)", nil),
		makeInputField("timeout", "Recording Timeout", "Examples: 30s, 2m, 5m.", cfg.Timeout.String(), "5m", func(s string) error {
			if _, err := time.ParseDuration(s); err != nil {
				return fmt.Errorf("invalid duration format")
			}
			return nil
		}),
	}
	screen := newFormScreen(state, "Recording Settings", nil, fields, func(values map[string]string) screen {
		state.cfg.Recording.SampleRate, _ = strconv.Atoi(values["sample_rate"])
		state.cfg.Recording.Channels, _ = strconv.Atoi(values["channels"])
		state.cfg.Recording.Format = values["format"]
		state.cfg.Recording.BufferSize, _ = strconv.Atoi(values["buffer_size"])
		state.cfg.Recording.ChannelBufferSize, _ = strconv.Atoi(values["channel_buffer"])
		state.cfg.Recording.Device = values["device"]
		state.cfg.Recording.Timeout, _ = time.ParseDuration(values["timeout"])
		return onBack()
	}, onBack)
	screen.footer = "enter save • esc back"
	return screen
}

func newInjectionTimeoutsScreen(state *wizardState, onBack func() screen) screen {
	cfg := state.cfg.Injection
	fields := []formField{
		makeInputField("ydotool", "ydotool Timeout", "Examples: 5s, 10s.", cfg.YdotoolTimeout.String(), "5s", func(s string) error {
			if _, err := time.ParseDuration(s); err != nil {
				return fmt.Errorf("invalid duration format")
			}
			return nil
		}),
		makeInputField("wtype", "wtype Timeout", "Examples: 5s, 10s.", cfg.WtypeTimeout.String(), "5s", func(s string) error {
			if _, err := time.ParseDuration(s); err != nil {
				return fmt.Errorf("invalid duration format")
			}
			return nil
		}),
		makeInputField("clipboard", "Clipboard Timeout", "Examples: 3s, 5s.", cfg.ClipboardTimeout.String(), "3s", func(s string) error {
			if _, err := time.ParseDuration(s); err != nil {
				return fmt.Errorf("invalid duration format")
			}
			return nil
		}),
	}
	screen := newFormScreen(state, "Injection Timeouts", nil, fields, func(values map[string]string) screen {
		state.cfg.Injection.YdotoolTimeout, _ = time.ParseDuration(values["ydotool"])
		state.cfg.Injection.WtypeTimeout, _ = time.ParseDuration(values["wtype"])
		state.cfg.Injection.ClipboardTimeout, _ = time.ParseDuration(values["clipboard"])
		return onBack()
	}, onBack)
	screen.footer = "enter save • esc back"
	return screen
}

func newSummaryScreen(state *wizardState, onBack func() screen) screen {
	summary := buildSummaryLines(state.cfg)
	items := []optionItem{
		{title: "Save", value: "save"},
		{title: "Cancel", value: "cancel"},
	}
	desc := []string{}
	desc = append(desc, summary...)
	screen := &summaryScreen{
		state:  state,
		title:  "Configuration Summary",
		desc:   desc,
		list:   newSummaryList(items),
		onSave: func() screen { state.result = &ConfigureResult{Config: state.cfg, Cancelled: false}; return nil },
		onBack: onBack,
	}
	screen.footer = "enter save • esc back"
	return screen
}

func buildVoiceProviderOptions(cfg *config.Config) []optionItem {
	var options []optionItem

	whisperStatus := deps.CheckWhisperCli()
	if whisperStatus.Installed {
		options = append(options, optionItem{title: "Whisper.cpp (local, no API key)", value: "whisper-cpp"})
	} else {
		options = append(options, optionItem{title: "Whisper.cpp (local, install required)", value: "whisper-cpp-disabled"})
	}

	configured := getConfiguredProviders(cfg)
	for _, name := range configured {
		p := provider.GetProvider(name)
		if p != nil && len(provider.ModelsOfType(p, provider.Transcription)) > 0 {
			switch name {
			case "openai":
				options = append(options, optionItem{title: "OpenAI Whisper", value: "openai"})
			case "groq":
				options = append(options,
					optionItem{title: "Groq Whisper (transcription)", value: "groq-transcription"},
					optionItem{title: "Groq Whisper (translate to English)", value: "groq-translation"},
				)
			case "mistral":
				options = append(options, optionItem{title: "Mistral Voxtral", value: "mistral-transcription"})
			case "elevenlabs":
				options = append(options, optionItem{title: "ElevenLabs Scribe", value: "elevenlabs"})
			case "deepgram":
				options = append(options, optionItem{title: "Deepgram Nova", value: "deepgram"})
			}
		}
	}

	configuredSet := make(map[string]bool)
	for _, name := range configured {
		configuredSet[name] = true
	}

	if !configuredSet["openai"] {
		options = append(options, optionItem{title: "OpenAI Whisper (add API key)", value: "openai"})
	}
	if !configuredSet["groq"] {
		options = append(options,
			optionItem{title: "Groq Whisper transcription (add API key)", value: "groq-transcription"},
			optionItem{title: "Groq Whisper translation (add API key)", value: "groq-translation"},
		)
	}
	if !configuredSet["mistral"] {
		options = append(options, optionItem{title: "Mistral Voxtral (add API key)", value: "mistral-transcription"})
	}
	if !configuredSet["elevenlabs"] {
		options = append(options, optionItem{title: "ElevenLabs Scribe (add API key)", value: "elevenlabs"})
	}
	if !configuredSet["deepgram"] {
		options = append(options, optionItem{title: "Deepgram Nova (add API key)", value: "deepgram"})
	}

	return options
}

func buildLLMProviderOptions(cfg *config.Config) []optionItem {
	var options []optionItem
	configured := getConfiguredProviders(cfg)
	for _, name := range configured {
		p := provider.GetProvider(name)
		if p != nil && len(provider.ModelsOfType(p, provider.LLM)) > 0 {
			switch name {
			case "openai":
				options = append(options, optionItem{title: "OpenAI GPT", value: "openai"})
			case "groq":
				options = append(options, optionItem{title: "Groq Llama (fast)", value: "groq"})
			}
		}
	}

	configuredSet := make(map[string]bool)
	for _, name := range configured {
		configuredSet[name] = true
	}
	if !configuredSet["openai"] {
		options = append(options, optionItem{title: "OpenAI GPT (add API key)", value: "openai"})
	}
	if !configuredSet["groq"] {
		options = append(options, optionItem{title: "Groq Llama (add API key)", value: "groq"})
	}

	return options
}

func formatProviderOption(cfg *config.Config, name string) string {
	status := "(not configured)"
	if pc, exists := cfg.Providers[name]; exists && pc.APIKey != "" {
		status = "(configured)"
	}

	switch name {
	case "openai":
		return fmt.Sprintf("OpenAI - Whisper + GPT %s", status)
	case "groq":
		return fmt.Sprintf("Groq - Whisper + Llama %s", status)
	case "mistral":
		return fmt.Sprintf("Mistral - Voxtral %s", status)
	case "elevenlabs":
		return fmt.Sprintf("ElevenLabs - Scribe %s", status)
	case "deepgram":
		return fmt.Sprintf("Deepgram - Nova %s", status)
	default:
		return fmt.Sprintf("%s %s", name, status)
	}
}

func formatProvidersLabel(cfg *config.Config) string {
	count := len(getConfiguredProviders(cfg))
	if count == 0 {
		return "Providers (none)"
	}
	return fmt.Sprintf("Providers (%d configured)", count)
}

func formatVoiceModelLabel(cfg *config.Config) string {
	if cfg.Transcription.Provider == "" || cfg.Transcription.Model == "" {
		return "Voice Model (not set)"
	}
	return fmt.Sprintf("Voice Model (%s/%s)", cfg.Transcription.Provider, cfg.Transcription.Model)
}

func formatLLMLabel(cfg *config.Config) string {
	if !cfg.LLM.Enabled {
		return "LLM (disabled)"
	}
	if cfg.LLM.Provider == "" || cfg.LLM.Model == "" {
		return "LLM (enabled)"
	}
	return fmt.Sprintf("LLM (%s/%s)", cfg.LLM.Provider, cfg.LLM.Model)
}

func formatKeywordsLabel(cfg *config.Config) string {
	if len(cfg.Keywords) == 0 {
		return "Keywords (none)"
	}
	return fmt.Sprintf("Keywords (%d)", len(cfg.Keywords))
}

func formatInjectionLabel(cfg *config.Config) string {
	if len(cfg.Injection.Backends) == 0 {
		return "Injection (none)"
	}
	return fmt.Sprintf("Injection (%s)", strings.Join(cfg.Injection.Backends, " -> "))
}

func formatNotificationsLabel(cfg *config.Config) string {
	if !cfg.Notifications.Enabled {
		return "Notifications (disabled)"
	}
	if cfg.Notifications.Type == "" {
		return "Notifications (enabled)"
	}
	return fmt.Sprintf("Notifications (%s)", cfg.Notifications.Type)
}

func formatAdvancedRecordingLabel(cfg *config.Config) string {
	return fmt.Sprintf("Recording Settings (rate=%d, timeout=%s)", cfg.Recording.SampleRate, cfg.Recording.Timeout)
}

func formatAdvancedInjectionTimeoutLabel(cfg *config.Config) string {
	return fmt.Sprintf("Injection Timeouts (ydotool=%s, wtype=%s, clipboard=%s)", cfg.Injection.YdotoolTimeout, cfg.Injection.WtypeTimeout, cfg.Injection.ClipboardTimeout)
}

func getNotificationMessage(cfg *config.Config, def notify.MessageDef) (string, string) {
	switch def.ConfigKey {
	case "recording_started":
		return cfg.Notifications.Messages.RecordingStarted.Title, cfg.Notifications.Messages.RecordingStarted.Body
	case "transcribing":
		return cfg.Notifications.Messages.Transcribing.Title, cfg.Notifications.Messages.Transcribing.Body
	case "llm_processing":
		return cfg.Notifications.Messages.LLMProcessing.Title, cfg.Notifications.Messages.LLMProcessing.Body
	case "config_reloaded":
		return cfg.Notifications.Messages.ConfigReloaded.Title, cfg.Notifications.Messages.ConfigReloaded.Body
	case "operation_cancelled":
		return cfg.Notifications.Messages.OperationCancelled.Title, cfg.Notifications.Messages.OperationCancelled.Body
	case "recording_aborted":
		return cfg.Notifications.Messages.RecordingAborted.Title, cfg.Notifications.Messages.RecordingAborted.Body
	case "injection_aborted":
		return cfg.Notifications.Messages.InjectionAborted.Title, cfg.Notifications.Messages.InjectionAborted.Body
	default:
		return "", ""
	}
}

func setNotificationMessage(cfg *config.Config, configKey string, msg config.MessageConfig) {
	switch configKey {
	case "recording_started":
		cfg.Notifications.Messages.RecordingStarted = msg
	case "transcribing":
		cfg.Notifications.Messages.Transcribing = msg
	case "llm_processing":
		cfg.Notifications.Messages.LLMProcessing = msg
	case "config_reloaded":
		cfg.Notifications.Messages.ConfigReloaded = msg
	case "operation_cancelled":
		cfg.Notifications.Messages.OperationCancelled = msg
	case "recording_aborted":
		cfg.Notifications.Messages.RecordingAborted = msg
	case "injection_aborted":
		cfg.Notifications.Messages.InjectionAborted = msg
	}
}

func findMessageDef(key string) *notify.MessageDef {
	for _, def := range notify.MessageDefs {
		if def.ConfigKey == key {
			return &def
		}
	}
	return nil
}

func buildSummaryLines(cfg *config.Config) []string {
	var lines []string

	providers := getConfiguredProviders(cfg)
	providerSummary := "none"
	if len(providers) > 0 {
		providerSummary = strings.Join(providers, ", ")
	}
	lines = append(lines, fmt.Sprintf("Providers: %s", providerSummary))

	lang := cfg.Transcription.Language
	if lang == "" {
		lang = "auto-detect"
	}
	voiceSummary := "not set"
	if cfg.Transcription.Provider != "" && cfg.Transcription.Model != "" {
		voiceSummary = fmt.Sprintf("%s/%s (%s)", cfg.Transcription.Provider, cfg.Transcription.Model, lang)
	}
	lines = append(lines, fmt.Sprintf("Voice Model: %s", voiceSummary))

	if cfg.LLM.Enabled {
		lines = append(lines, fmt.Sprintf("LLM: %s (%s)", cfg.LLM.Provider, cfg.LLM.Model))
		var opts []string
		if cfg.LLM.PostProcessing.RemoveStutters {
			opts = append(opts, "remove stutters")
		}
		if cfg.LLM.PostProcessing.AddPunctuation {
			opts = append(opts, "add punctuation")
		}
		if cfg.LLM.PostProcessing.FixGrammar {
			opts = append(opts, "fix grammar")
		}
		if cfg.LLM.PostProcessing.RemoveFillerWords {
			opts = append(opts, "remove fillers")
		}
		if len(opts) > 0 {
			lines = append(lines, fmt.Sprintf("Post-processing: %s", strings.Join(opts, ", ")))
		}
	} else {
		lines = append(lines, "LLM: disabled")
	}

	if len(cfg.Keywords) > 0 {
		lines = append(lines, fmt.Sprintf("Keywords: %s", strings.Join(cfg.Keywords, ", ")))
	}

	backendSummary := "none"
	if len(cfg.Injection.Backends) > 0 {
		backendSummary = strings.Join(cfg.Injection.Backends, " -> ")
	}
	lines = append(lines, fmt.Sprintf("Backends: %s", backendSummary))

	notifSummary := "disabled"
	if cfg.Notifications.Enabled {
		if cfg.Notifications.Type != "" {
			notifSummary = fmt.Sprintf("enabled (%s)", cfg.Notifications.Type)
		} else {
			notifSummary = "enabled"
		}
	}
	lines = append(lines, fmt.Sprintf("Notifications: %s", notifSummary))

	return lines
}

func selectListByValue(l *list.Model, value string) {
	if value == "" {
		return
	}
	items := l.Items()
	for i, item := range items {
		switch v := item.(type) {
		case optionItem:
			if v.value == value {
				l.Select(i)
				return
			}
		case toggleItem:
			if v.value == value {
				l.Select(i)
				return
			}
		}
	}
}

func newSummaryList(items []optionItem) list.Model {
	delegate := list.NewDefaultDelegate()
	l := list.New(itemsToList(items), delegate, 0, 0)
	l.DisableQuitKeybindings()
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetShowStatusBar(false)
	return l
}

type summaryScreen struct {
	state  *wizardState
	title  string
	desc   []string
	list   list.Model
	footer string
	onSave func() screen
	onBack func() screen
}

func (s *summaryScreen) Init() tea.Cmd { return nil }

func (s *summaryScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.list.SetSize(msg.Width-4, msg.Height-10)
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := s.list.SelectedItem().(optionItem); ok {
				if item.value == "save" && s.onSave != nil {
					return s.onSave(), nil
				}
				if item.value == "cancel" && s.onBack != nil {
					return s.onBack(), nil
				}
			}
		case "esc", "q":
			if s.onBack != nil {
				return s.onBack(), nil
			}
		}
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *summaryScreen) View() string {
	header := renderHeader(s.title, nil, "")
	var body strings.Builder
	for _, line := range s.desc {
		body.WriteString(StyleLabel.Render(line))
		body.WriteString("\n")
	}
	body.WriteString("\n")
	footer := renderFooter(s.footer, false)
	return header + body.String() + s.list.View() + "\n" + footer
}

func downloadWhisperModel(modelID string, onProgress func(downloaded, total int64)) error {
	return whisper.Download(context.Background(), modelID, onProgress)
}
