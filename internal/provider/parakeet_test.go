package provider

import "testing"

func TestParakeetProvider_GetProvider(t *testing.T) {
	p := GetProvider("parakeet")
	if p == nil {
		t.Fatal("GetProvider('parakeet') returned nil")
	}
	if p.Name() != "parakeet" {
		t.Errorf("expected name 'parakeet', got '%s'", p.Name())
	}
}

func TestParakeetProvider_Models(t *testing.T) {
	p := &ParakeetProvider{}
	models := p.Models()

	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	for _, m := range models {
		if m.ID != "parakeet-tdt-0.6b-v3" {
			t.Errorf("expected model ID 'parakeet-tdt-0.6b-v3', got '%s'", m.ID)
		}
		if !m.Local {
			t.Errorf("model %s: expected Local=true", m.ID)
		}
		if m.LocalInfo == nil {
			t.Errorf("model %s: expected LocalInfo to be set", m.ID)
		}
		if m.AdapterType != "parakeet" {
			t.Errorf("model %s: expected AdapterType='parakeet', got '%s'", m.ID, m.AdapterType)
		}
		if m.Type != Transcription {
			t.Errorf("model %s: expected Type=Transcription", m.ID)
		}
		if m.Endpoint != nil {
			t.Errorf("model %s: expected Endpoint=nil for local model", m.ID)
		}
		if m.LocalInfo.Filename == "" || m.LocalInfo.Size == "" || m.LocalInfo.DownloadURL == "" {
			t.Errorf("model %s: LocalInfo fields should not be empty", m.ID)
		}
		if !m.NeedsDownload() {
			t.Errorf("model %s: NeedsDownload() should be true for local model", m.ID)
		}
	}
}

func TestParakeetProvider_LanguageSupport(t *testing.T) {
	p := &ParakeetProvider{}
	m := p.Models()[0]

	for _, lang := range []string{"en", "es", "fr", "de", "it", "pl", "ru"} {
		if !m.SupportsLanguage(lang) {
			t.Errorf("model %s: SupportsLanguage(%q) should be true", m.ID, lang)
		}
	}
	if !m.SupportsLanguage("") {
		t.Errorf("model %s: SupportsLanguage('') should be true (auto always supported)", m.ID)
	}
	if m.SupportsLanguage("ja") {
		t.Errorf("model %s: SupportsLanguage('ja') should be false (not a European language)", m.ID)
	}
}

func TestParakeetProvider_RequiresAPIKey(t *testing.T) {
	p := &ParakeetProvider{}
	if p.RequiresAPIKey() {
		t.Error("RequiresAPIKey() should return false")
	}
}

func TestParakeetProvider_IsLocal(t *testing.T) {
	p := &ParakeetProvider{}
	if !p.IsLocal() {
		t.Error("IsLocal() should return true")
	}
}

func TestParakeetProvider_DefaultModel(t *testing.T) {
	p := &ParakeetProvider{}
	if p.DefaultModel(Transcription) != "parakeet-tdt-0.6b-v3" {
		t.Errorf("expected DefaultModel(Transcription)='parakeet-tdt-0.6b-v3', got '%s'", p.DefaultModel(Transcription))
	}
	if p.DefaultModel(LLM) != "" {
		t.Errorf("expected DefaultModel(LLM)='', got '%s'", p.DefaultModel(LLM))
	}
}
