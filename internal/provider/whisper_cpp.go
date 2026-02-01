package provider

import (
	"github.com/leonardotrapani/hyprvoice/internal/language"
	"github.com/leonardotrapani/hyprvoice/internal/models/whisper"
)

// WhisperCppProvider implements Provider for local whisper.cpp transcription
type WhisperCppProvider struct{}

func (p *WhisperCppProvider) Name() string {
	return ProviderWhisperCpp
}

func (p *WhisperCppProvider) RequiresAPIKey() bool {
	return false
}

func (p *WhisperCppProvider) ValidateAPIKey(key string) bool {
	return true // no API key needed
}

func (p *WhisperCppProvider) IsLocal() bool {
	return true
}

func (p *WhisperCppProvider) Models() []Model {
	allLangs := language.AllLanguageCodes()
	englishOnly := []string{"en"}
	docsURL := "https://github.com/openai/whisper#available-models-and-languages"

	whisperModels := whisper.ListModels()
	result := make([]Model, 0, len(whisperModels))

	for _, wm := range whisperModels {
		var langs []string
		if wm.Multilingual {
			langs = allLangs
		} else {
			langs = englishOnly
		}

		result = append(result, Model{
			ID:                 wm.ID,
			Name:               wm.Name,
			Description:        modelDescription(wm),
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              true,
			AdapterType:        AdapterWhisperCpp,
			SupportedLanguages: langs,
			Endpoint:           nil, // local CLI, no HTTP endpoint
			LocalInfo: &LocalModelInfo{
				Filename:    wm.Filename,
				Size:        wm.Size,
				DownloadURL: whisper.GetDownloadURL(wm.ID),
			},
			DocsURL: docsURL,
		})
	}

	return result
}

func modelDescription(m whisper.ModelInfo) string {
	if m.Multilingual {
		return "Multilingual local transcription"
	}
	return "English-only local transcription (faster)"
}

func (p *WhisperCppProvider) DefaultModel(t ModelType) string {
	switch t {
	case Transcription:
		return "base.en"
	}
	return ""
}
