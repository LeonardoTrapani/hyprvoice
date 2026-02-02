package provider

import "testing"

func TestDeepgramProvider(t *testing.T) {
	p := GetProvider("deepgram")
	if p == nil {
		t.Fatal("deepgram provider not registered")
	}

	if p.Name() != "deepgram" {
		t.Errorf("Name() = %q, want %q", p.Name(), "deepgram")
	}

	if !p.RequiresAPIKey() {
		t.Error("RequiresAPIKey() should return true")
	}

	if p.IsLocal() {
		t.Error("IsLocal() should return false")
	}
}

func TestDeepgramProvider_Models(t *testing.T) {
	p := &DeepgramProvider{}
	models := p.Models()

	if len(models) != 2 {
		t.Errorf("Models() returned %d models, want 2", len(models))
	}

	// all models should support both streaming and batch
	for _, m := range models {
		if !m.SupportsStreaming {
			t.Errorf("model %s should support streaming", m.ID)
		}
		if !m.SupportsBothModes() {
			t.Errorf("model %s should support both modes", m.ID)
		}
		if m.AdapterType != "deepgram" {
			t.Errorf("model %s has AdapterType %q, want 'deepgram'", m.ID, m.AdapterType)
		}
		if m.Local {
			t.Errorf("model %s should not be local", m.ID)
		}
	}
}

func TestDeepgramProvider_Nova3Languages(t *testing.T) {
	p := &DeepgramProvider{}
	models := p.Models()

	var nova3 *Model
	for i := range models {
		if models[i].ID == "nova-3" {
			nova3 = &models[i]
			break
		}
	}
	if nova3 == nil {
		t.Fatal("nova-3 model not found")
	}

	// nova-3 should support many languages from our list
	supportedTests := []struct {
		code string
		want bool
	}{
		{"en", true},
		{"es", true},
		{"fr", true},
		{"de", true},
		{"ja", true},
		{"", true}, // auto always supported
	}

	for _, tt := range supportedTests {
		got := nova3.SupportsLanguage(tt.code)
		if got != tt.want {
			t.Errorf("nova-3.SupportsLanguage(%q) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestDeepgramProvider_DefaultModel(t *testing.T) {
	p := &DeepgramProvider{}

	if got := p.DefaultModel(Transcription); got != "nova-3" {
		t.Errorf("DefaultModel(Transcription) = %q, want 'nova-3'", got)
	}

	if got := p.DefaultModel(LLM); got != "" {
		t.Errorf("DefaultModel(LLM) = %q, want empty (no LLM support)", got)
	}
}

func TestDeepgramProvider_Endpoint(t *testing.T) {
	p := &DeepgramProvider{}
	models := p.Models()

	for _, m := range models {
		// batch endpoint (HTTP)
		if m.Endpoint == nil {
			t.Errorf("model %s has nil Endpoint", m.ID)
			continue
		}
		if m.Endpoint.BaseURL != "https://api.deepgram.com" {
			t.Errorf("model %s has Endpoint.BaseURL %q, want 'https://api.deepgram.com'", m.ID, m.Endpoint.BaseURL)
		}
		if m.Endpoint.Path != "/v1/listen" {
			t.Errorf("model %s has Endpoint.Path %q, want '/v1/listen'", m.ID, m.Endpoint.Path)
		}

		// streaming endpoint (WebSocket)
		if m.StreamingEndpoint == nil {
			t.Errorf("model %s has nil StreamingEndpoint", m.ID)
			continue
		}
		if m.StreamingEndpoint.BaseURL != "wss://api.deepgram.com" {
			t.Errorf("model %s has StreamingEndpoint.BaseURL %q, want 'wss://api.deepgram.com'", m.ID, m.StreamingEndpoint.BaseURL)
		}
		if m.StreamingEndpoint.Path != "/v1/listen" {
			t.Errorf("model %s has StreamingEndpoint.Path %q, want '/v1/listen'", m.ID, m.StreamingEndpoint.Path)
		}
	}
}
