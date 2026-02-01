package provider

import (
	"testing"

	"github.com/leonardotrapani/hyprvoice/internal/language"
)

func TestModel_NeedsDownload(t *testing.T) {
	tests := []struct {
		name     string
		model    Model
		expected bool
	}{
		{
			name: "local model with LocalInfo",
			model: Model{
				ID:    "base.en",
				Local: true,
				LocalInfo: &LocalModelInfo{
					Filename:    "ggml-base.en.bin",
					Size:        "142MB",
					DownloadURL: "https://example.com/model.bin",
				},
			},
			expected: true,
		},
		{
			name: "cloud model without LocalInfo",
			model: Model{
				ID:       "whisper-1",
				Local:    false,
				Endpoint: &EndpointConfig{BaseURL: "https://api.openai.com", Path: "/v1/audio/transcriptions"},
			},
			expected: false,
		},
		{
			name: "model with nil LocalInfo",
			model: Model{
				ID:        "gpt-4o",
				LocalInfo: nil,
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.model.NeedsDownload(); got != tc.expected {
				t.Errorf("NeedsDownload() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestModel_IsStreaming(t *testing.T) {
	tests := []struct {
		name     string
		model    Model
		expected bool
	}{
		{
			name:     "streaming model",
			model:    Model{ID: "scribe_v1-streaming", Streaming: true},
			expected: true,
		},
		{
			name:     "batch model",
			model:    Model{ID: "whisper-1", Streaming: false},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.model.IsStreaming(); got != tc.expected {
				t.Errorf("IsStreaming() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestModel_SupportsLanguage(t *testing.T) {
	allCodes := language.AllLanguageCodes()

	multilingualModel := Model{
		ID:                 "whisper-large-v3",
		SupportedLanguages: allCodes,
	}

	englishOnlyModel := Model{
		ID:                 "base.en",
		SupportedLanguages: []string{"en"},
	}

	tests := []struct {
		name     string
		model    Model
		code     string
		expected bool
	}{
		{
			name:     "multilingual supports en",
			model:    multilingualModel,
			code:     "en",
			expected: true,
		},
		{
			name:     "multilingual supports es",
			model:    multilingualModel,
			code:     "es",
			expected: true,
		},
		{
			name:     "multilingual supports zh",
			model:    multilingualModel,
			code:     "zh",
			expected: true,
		},
		{
			name:     "english-only supports en",
			model:    englishOnlyModel,
			code:     "en",
			expected: true,
		},
		{
			name:     "english-only does not support es",
			model:    englishOnlyModel,
			code:     "es",
			expected: false,
		},
		{
			name:     "english-only does not support zh",
			model:    englishOnlyModel,
			code:     "zh",
			expected: false,
		},
		{
			name:     "auto always supported on multilingual",
			model:    multilingualModel,
			code:     "",
			expected: true,
		},
		{
			name:     "auto always supported on english-only",
			model:    englishOnlyModel,
			code:     "",
			expected: true,
		},
		{
			name:     "empty SupportedLanguages still supports auto",
			model:    Model{ID: "empty", SupportedLanguages: []string{}},
			code:     "",
			expected: true,
		},
		{
			name:     "empty SupportedLanguages does not support en",
			model:    Model{ID: "empty", SupportedLanguages: []string{}},
			code:     "en",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.model.SupportsLanguage(tc.code); got != tc.expected {
				t.Errorf("SupportsLanguage(%q) = %v, want %v", tc.code, got, tc.expected)
			}
		})
	}
}

func TestModel_SupportsAllLanguages(t *testing.T) {
	allCodes := language.AllLanguageCodes()

	tests := []struct {
		name     string
		model    Model
		expected bool
	}{
		{
			name: "model with all 57 languages",
			model: Model{
				ID:                 "whisper-large-v3",
				SupportedLanguages: allCodes,
			},
			expected: true,
		},
		{
			name: "english-only model",
			model: Model{
				ID:                 "base.en",
				SupportedLanguages: []string{"en"},
			},
			expected: false,
		},
		{
			name: "model with some languages",
			model: Model{
				ID:                 "partial",
				SupportedLanguages: []string{"en", "es", "fr", "de"},
			},
			expected: false,
		},
		{
			name: "model with empty languages",
			model: Model{
				ID:                 "empty",
				SupportedLanguages: []string{},
			},
			expected: false,
		},
		{
			name: "model with nil languages",
			model: Model{
				ID:                 "nil",
				SupportedLanguages: nil,
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.model.SupportsAllLanguages(); got != tc.expected {
				t.Errorf("SupportsAllLanguages() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestModelType_Constants(t *testing.T) {
	// verify ModelType constants exist and are distinct
	if Transcription == LLM {
		t.Error("Transcription and LLM should be different")
	}

	// verify they're the expected values
	if Transcription != 0 {
		t.Errorf("Transcription = %d, want 0", Transcription)
	}
	if LLM != 1 {
		t.Errorf("LLM = %d, want 1", LLM)
	}
}

func TestEndpointConfig_Fields(t *testing.T) {
	endpoint := EndpointConfig{
		BaseURL: "https://api.openai.com",
		Path:    "/v1/audio/transcriptions",
	}

	if endpoint.BaseURL != "https://api.openai.com" {
		t.Errorf("BaseURL = %q, want 'https://api.openai.com'", endpoint.BaseURL)
	}
	if endpoint.Path != "/v1/audio/transcriptions" {
		t.Errorf("Path = %q, want '/v1/audio/transcriptions'", endpoint.Path)
	}
}

func TestLocalModelInfo_Fields(t *testing.T) {
	info := LocalModelInfo{
		Filename:    "ggml-base.en.bin",
		Size:        "142MB",
		DownloadURL: "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin",
	}

	if info.Filename != "ggml-base.en.bin" {
		t.Errorf("Filename = %q, want 'ggml-base.en.bin'", info.Filename)
	}
	if info.Size != "142MB" {
		t.Errorf("Size = %q, want '142MB'", info.Size)
	}
	if info.DownloadURL != "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin" {
		t.Errorf("DownloadURL = %q", info.DownloadURL)
	}
}

func TestModel_AllFields(t *testing.T) {
	// verify all Model struct fields can be set and read correctly
	model := Model{
		ID:                 "test-model",
		Name:               "Test Model",
		Description:        "A test model for verification",
		Type:               Transcription,
		Streaming:          true,
		Local:              true,
		AdapterType:        "test-adapter",
		SupportedLanguages: []string{"en", "es"},
		Endpoint: &EndpointConfig{
			BaseURL: "https://api.test.com",
			Path:    "/v1/test",
		},
		LocalInfo: &LocalModelInfo{
			Filename:    "test.bin",
			Size:        "100MB",
			DownloadURL: "https://example.com/test.bin",
		},
		DocsURL: "https://example.com/docs/languages",
	}

	if model.ID != "test-model" {
		t.Errorf("ID = %q, want 'test-model'", model.ID)
	}
	if model.Name != "Test Model" {
		t.Errorf("Name = %q, want 'Test Model'", model.Name)
	}
	if model.Description != "A test model for verification" {
		t.Errorf("Description = %q", model.Description)
	}
	if model.Type != Transcription {
		t.Errorf("Type = %v, want Transcription", model.Type)
	}
	if !model.Streaming {
		t.Error("Streaming should be true")
	}
	if !model.Local {
		t.Error("Local should be true")
	}
	if model.AdapterType != "test-adapter" {
		t.Errorf("AdapterType = %q, want 'test-adapter'", model.AdapterType)
	}
	if len(model.SupportedLanguages) != 2 {
		t.Errorf("SupportedLanguages length = %d, want 2", len(model.SupportedLanguages))
	}
	if model.Endpoint == nil {
		t.Error("Endpoint should not be nil")
	}
	if model.LocalInfo == nil {
		t.Error("LocalInfo should not be nil")
	}
	if model.DocsURL != "https://example.com/docs/languages" {
		t.Errorf("DocsURL = %q, want 'https://example.com/docs/languages'", model.DocsURL)
	}
}

func TestAllTranscriptionModels_HaveDocsURL(t *testing.T) {
	// verify all transcription models have DocsURL set
	providers := []string{"openai", "groq", "mistral", "elevenlabs", "deepgram", "whisper-cpp"}

	expectedDocsURLs := map[string]string{
		"openai":      "https://platform.openai.com/docs/guides/speech-to-text#supported-languages",
		"groq":        "https://console.groq.com/docs/speech-to-text#supported-languages",
		"mistral":     "https://docs.mistral.ai/capabilities/speech/",
		"elevenlabs":  "https://elevenlabs.io/docs/capabilities/speech-to-text#supported-languages",
		"deepgram":    "https://developers.deepgram.com/docs/language",
		"whisper-cpp": "https://github.com/openai/whisper#available-models-and-languages",
	}

	for _, pName := range providers {
		p := GetProvider(pName)
		if p == nil {
			t.Errorf("GetProvider(%q) returned nil", pName)
			continue
		}

		expectedURL := expectedDocsURLs[pName]
		for _, m := range p.Models() {
			if m.Type != Transcription {
				continue
			}
			if m.DocsURL == "" {
				t.Errorf("%s/%s: DocsURL is empty", pName, m.ID)
			} else if m.DocsURL != expectedURL {
				t.Errorf("%s/%s: DocsURL = %q, want %q", pName, m.ID, m.DocsURL, expectedURL)
			}
		}
	}
}
