package provider

import "github.com/leonardotrapani/hyprvoice/internal/models/whisper"

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

func (p *WhisperCppProvider) APIKeyURL() string {
	return ""
}

func (p *WhisperCppProvider) IsLocal() bool {
	return true
}

func (p *WhisperCppProvider) Models() []Model {
	// https://github.com/ggml-org/whisper.cpp#models
	allLangs := whisperTranscriptionLanguages
	// https://github.com/ggml-org/whisper.cpp#models
	englishOnly := whisperEnglishOnlyLanguages
	docsURL := "https://github.com/ggml-org/whisper.cpp#models"

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
	switch m.ID {
	case "tiny.en":
		return "Free/offline; fastest but low accuracy, good for weak hardware"
	case "base.en":
		return "Free/offline; balanced speed and accuracy, recommended start"
	case "small.en":
		return "Free/offline; better accuracy, needs decent CPU"
	case "medium.en":
		return "Free/offline; best .en accuracy, needs good CPU/RAM"
	case "tiny":
		return "Free/offline multilingual; fastest but low accuracy"
	case "base":
		return "Free/offline multilingual; balanced, recommended start"
	case "small":
		return "Free/offline multilingual; better accuracy, needs decent CPU"
	case "medium":
		return "Free/offline multilingual; great accuracy, needs good CPU/RAM"
	case "large-v1":
		return "Free/offline; high accuracy, needs strong CPU/GPU"
	case "large-v2":
		return "Free/offline; high accuracy, needs strong CPU/GPU"
	case "large-v3":
		return "Free/offline; best accuracy available, needs strong hardware"
	case "large-v3-turbo":
		return "Free/offline; near-best accuracy with better speed"
	}
	if m.Multilingual {
		return "Free/offline multilingual model"
	}
	return "Free/offline English model"
}

func (p *WhisperCppProvider) DefaultModel(t ModelType) string {
	switch t {
	case Transcription:
		return "base.en"
	}
	return ""
}
