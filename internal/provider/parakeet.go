package provider

import "github.com/leonardotrapani/hyprvoice/internal/models/parakeet"

// ParakeetProvider implements Provider for local NVIDIA Parakeet transcription
// via the NeMo-Speech.cpp runtime (nemo-speech CLI).
type ParakeetProvider struct{}

func (p *ParakeetProvider) Name() string {
	return ProviderParakeet
}

func (p *ParakeetProvider) RequiresAPIKey() bool {
	return false
}

func (p *ParakeetProvider) ValidateAPIKey(key string) bool {
	return true // no API key needed
}

func (p *ParakeetProvider) APIKeyURL() string {
	return ""
}

func (p *ParakeetProvider) IsLocal() bool {
	return true
}

func (p *ParakeetProvider) Models() []Model {
	// https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3
	docsURL := "https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3"

	parakeetModels := parakeet.ListModels()
	result := make([]Model, 0, len(parakeetModels))

	for _, pm := range parakeetModels {
		result = append(result, Model{
			ID:                 pm.ID,
			Name:               pm.Name,
			Description:        "Free/offline; NVIDIA Parakeet TDT, high accuracy, auto language detection (NeMo-Speech.cpp)",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:  false,
			Local:              true,
			AdapterType:        AdapterParakeet,
			SupportedLanguages: pm.Languages,
			Endpoint:           nil, // local CLI, no HTTP endpoint
			LocalInfo: &LocalModelInfo{
				Filename:    pm.Filename,
				Size:        pm.Size,
				DownloadURL: pm.DownloadURL,
			},
			DocsURL: docsURL,
		})
	}

	return result
}

func (p *ParakeetProvider) DefaultModel(t ModelType) string {
	switch t {
	case Transcription:
		return "parakeet-tdt-0.6b-v3"
	}
	return ""
}
