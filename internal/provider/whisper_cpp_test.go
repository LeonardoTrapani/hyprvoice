package provider

import "testing"

func TestWhisperCppProvider_GetProvider(t *testing.T) {
	p := GetProvider("whisper-cpp")
	if p == nil {
		t.Fatal("GetProvider('whisper-cpp') returned nil")
	}
	if p.Name() != "whisper-cpp" {
		t.Errorf("expected name 'whisper-cpp', got '%s'", p.Name())
	}
}

func TestWhisperCppProvider_Models(t *testing.T) {
	p := &WhisperCppProvider{}
	models := p.Models()

	// verify we have 12 models
	if len(models) != 12 {
		t.Errorf("expected 12 models, got %d", len(models))
	}

	// verify all models have required fields
	for _, m := range models {
		if !m.Local {
			t.Errorf("model %s: expected Local=true", m.ID)
		}
		if m.LocalInfo == nil {
			t.Errorf("model %s: expected LocalInfo to be set", m.ID)
		}
		if m.AdapterType != "whisper-cpp" {
			t.Errorf("model %s: expected AdapterType='whisper-cpp', got '%s'", m.ID, m.AdapterType)
		}
		if m.Type != Transcription {
			t.Errorf("model %s: expected Type=Transcription", m.ID)
		}
		if m.Endpoint != nil {
			t.Errorf("model %s: expected Endpoint=nil for local model", m.ID)
		}
	}
}

func TestWhisperCppProvider_EnglishOnlyModels(t *testing.T) {
	p := &WhisperCppProvider{}
	models := p.Models()

	englishOnlyIDs := map[string]bool{
		"tiny.en":   true,
		"base.en":   true,
		"small.en":  true,
		"medium.en": true,
	}

	for _, m := range models {
		isEnglishOnly := englishOnlyIDs[m.ID]
		if isEnglishOnly {
			// english-only models should only support 'en'
			if len(m.SupportedLanguages) != 1 || m.SupportedLanguages[0] != "en" {
				t.Errorf("model %s: expected SupportedLanguages=['en'], got %v", m.ID, m.SupportedLanguages)
			}
			if m.SupportsLanguage("es") {
				t.Errorf("model %s: SupportsLanguage('es') should be false", m.ID)
			}
			if !m.SupportsLanguage("en") {
				t.Errorf("model %s: SupportsLanguage('en') should be true", m.ID)
			}
			if !m.SupportsLanguage("") {
				t.Errorf("model %s: SupportsLanguage('') should be true (auto always supported)", m.ID)
			}
		}
	}
}

func TestWhisperCppProvider_MultilingualModels(t *testing.T) {
	p := &WhisperCppProvider{}
	models := p.Models()

	multilingualIDs := map[string]bool{
		"tiny":           true,
		"base":           true,
		"small":          true,
		"medium":         true,
		"large-v1":       true,
		"large-v2":       true,
		"large-v3":       true,
		"large-v3-turbo": true,
	}

	for _, m := range models {
		isMultilingual := multilingualIDs[m.ID]
		if isMultilingual {
			if len(m.SupportedLanguages) <= 1 {
				t.Errorf("model %s: expected multiple languages, got %d", m.ID, len(m.SupportedLanguages))
			}
			if !m.SupportsLanguage("es") {
				t.Errorf("model %s: SupportsLanguage('es') should be true", m.ID)
			}
			if !m.SupportsLanguage("en") {
				t.Errorf("model %s: SupportsLanguage('en') should be true", m.ID)
			}
		}
	}
}

func TestWhisperCppProvider_RequiresAPIKey(t *testing.T) {
	p := &WhisperCppProvider{}
	if p.RequiresAPIKey() {
		t.Error("RequiresAPIKey() should return false")
	}
}

func TestWhisperCppProvider_IsLocal(t *testing.T) {
	p := &WhisperCppProvider{}
	if !p.IsLocal() {
		t.Error("IsLocal() should return true")
	}
}

func TestWhisperCppProvider_DefaultModel(t *testing.T) {
	p := &WhisperCppProvider{}
	if p.DefaultModel(Transcription) != "base.en" {
		t.Errorf("expected DefaultModel(Transcription)='base.en', got '%s'", p.DefaultModel(Transcription))
	}
	if p.DefaultModel(LLM) != "" {
		t.Errorf("expected DefaultModel(LLM)='', got '%s'", p.DefaultModel(LLM))
	}
}

func TestWhisperCppProvider_LocalInfo(t *testing.T) {
	p := &WhisperCppProvider{}
	models := p.Models()

	for _, m := range models {
		if m.LocalInfo.Filename == "" {
			t.Errorf("model %s: LocalInfo.Filename should not be empty", m.ID)
		}
		if m.LocalInfo.Size == "" {
			t.Errorf("model %s: LocalInfo.Size should not be empty", m.ID)
		}
		if m.LocalInfo.DownloadURL == "" {
			t.Errorf("model %s: LocalInfo.DownloadURL should not be empty", m.ID)
		}
		if !m.NeedsDownload() {
			t.Errorf("model %s: NeedsDownload() should be true for local model", m.ID)
		}
	}
}
