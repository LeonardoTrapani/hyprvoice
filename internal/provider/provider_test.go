package provider

import (
	"slices"
	"strings"
	"testing"
)

func TestProviderInterface(t *testing.T) {
	providers := []struct {
		name              string
		hasTranscription  bool
		hasLLM            bool
		isLocal           bool
		defaultTransModel string
		defaultLLMModel   string
	}{
		{"openai", true, true, false, "whisper-1", "gpt-4o-mini"},
		{"groq", true, true, false, "whisper-large-v3-turbo", "llama-3.3-70b-versatile"},
		{"mistral", true, false, false, "voxtral-mini-latest", ""},
		{"elevenlabs", true, false, false, "scribe_v1", ""},
	}

	for _, tc := range providers {
		t.Run(tc.name, func(t *testing.T) {
			p := GetProvider(tc.name)
			if p == nil {
				t.Fatalf("GetProvider(%q) returned nil", tc.name)
			}

			if p.Name() != tc.name {
				t.Errorf("Name() = %q, want %q", p.Name(), tc.name)
			}

			hasTranscription := len(ModelsOfType(p, Transcription)) > 0
			if hasTranscription != tc.hasTranscription {
				t.Errorf("hasTranscription = %v, want %v", hasTranscription, tc.hasTranscription)
			}

			hasLLM := len(ModelsOfType(p, LLM)) > 0
			if hasLLM != tc.hasLLM {
				t.Errorf("hasLLM = %v, want %v", hasLLM, tc.hasLLM)
			}

			if p.IsLocal() != tc.isLocal {
				t.Errorf("IsLocal() = %v, want %v", p.IsLocal(), tc.isLocal)
			}

			if p.DefaultModel(Transcription) != tc.defaultTransModel {
				t.Errorf("DefaultModel(Transcription) = %q, want %q", p.DefaultModel(Transcription), tc.defaultTransModel)
			}

			if p.DefaultModel(LLM) != tc.defaultLLMModel {
				t.Errorf("DefaultModel(LLM) = %q, want %q", p.DefaultModel(LLM), tc.defaultLLMModel)
			}

			if !p.RequiresAPIKey() {
				t.Error("RequiresAPIKey() should be true for all cloud providers")
			}

			if tc.hasTranscription && len(ModelsOfType(p, Transcription)) == 0 {
				t.Error("should have transcription models")
			}

			if tc.hasLLM && len(ModelsOfType(p, LLM)) == 0 {
				t.Error("should have LLM models")
			}
		})
	}
}

func TestGetProviderNotFound(t *testing.T) {
	p := GetProvider("nonexistent")
	if p != nil {
		t.Errorf("GetProvider(nonexistent) should return nil, got %v", p)
	}
}

func TestListProviders(t *testing.T) {
	providers := ListProviders()
	expected := []string{"openai", "groq", "mistral", "elevenlabs"}

	for _, name := range expected {
		if !slices.Contains(providers, name) {
			t.Errorf("ListProviders() missing %q", name)
		}
	}
}

func TestListProvidersWithTranscription(t *testing.T) {
	providers := ListProvidersWithTranscription()
	// All providers support transcription
	expected := []string{"openai", "groq", "mistral", "elevenlabs"}

	for _, name := range expected {
		if !slices.Contains(providers, name) {
			t.Errorf("ListProvidersWithTranscription() missing %q", name)
		}
	}
}

func TestListProvidersWithLLM(t *testing.T) {
	providers := ListProvidersWithLLM()
	expected := []string{"openai", "groq"}

	for _, name := range expected {
		if !slices.Contains(providers, name) {
			t.Errorf("ListProvidersWithLLM() missing %q", name)
		}
	}

	// Mistral and ElevenLabs should NOT be in the list
	notExpected := []string{"mistral", "elevenlabs"}
	for _, name := range notExpected {
		if slices.Contains(providers, name) {
			t.Errorf("ListProvidersWithLLM() should not include %q", name)
		}
	}
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		provider string
		key      string
		valid    bool
	}{
		{"openai", "sk-abc123", true},
		{"openai", "invalid", false},
		{"openai", "", false},
		{"groq", "gsk_abc123", true},
		{"groq", "invalid", false},
		{"groq", "", false},
		{"mistral", "any-non-empty", true},
		{"mistral", "", false},
		{"elevenlabs", "any-non-empty", true},
		{"elevenlabs", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.provider+"_"+tc.key, func(t *testing.T) {
			p := GetProvider(tc.provider)
			if p.ValidateAPIKey(tc.key) != tc.valid {
				t.Errorf("ValidateAPIKey(%q) = %v, want %v", tc.key, !tc.valid, tc.valid)
			}
		})
	}
}

func TestGetModel(t *testing.T) {
	// valid provider and model
	m, err := GetModel("openai", "whisper-1")
	if err != nil {
		t.Errorf("GetModel('openai', 'whisper-1') unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("GetModel returned nil model")
	}
	if m.ID != "whisper-1" {
		t.Errorf("GetModel returned model with ID %q, want 'whisper-1'", m.ID)
	}

	// unknown provider
	_, err = GetModel("nonexistent", "whisper-1")
	if err == nil {
		t.Error("GetModel('nonexistent', ...) should return error")
	}

	// unknown model
	_, err = GetModel("openai", "nonexistent")
	if err == nil {
		t.Error("GetModel('openai', 'nonexistent') should return error")
	}
}

func TestModelsOfType(t *testing.T) {
	p := GetProvider("openai")
	trans := ModelsOfType(p, Transcription)
	llm := ModelsOfType(p, LLM)

	// OpenAI has 3 transcription models: whisper-1, gpt-4o-transcribe, gpt-4o-mini-transcribe
	if len(trans) != 3 {
		t.Errorf("ModelsOfType(Transcription) = %d, want 3", len(trans))
	}
	// OpenAI has 2 LLM models: gpt-4o-mini, gpt-4o
	if len(llm) != 2 {
		t.Errorf("ModelsOfType(LLM) = %d, want 2", len(llm))
	}
}

func TestFindModelByID(t *testing.T) {
	// find model that exists
	m, p, err := FindModelByID("whisper-1")
	if err != nil {
		t.Errorf("FindModelByID('whisper-1') unexpected error: %v", err)
	}
	if m == nil || p == nil {
		t.Fatal("FindModelByID returned nil")
	}
	if m.ID != "whisper-1" {
		t.Errorf("FindModelByID returned model %q, want 'whisper-1'", m.ID)
	}
	if p.Name() != "openai" {
		t.Errorf("FindModelByID returned provider %q, want 'openai'", p.Name())
	}

	// model not found
	_, _, err = FindModelByID("nonexistent")
	if err == nil {
		t.Error("FindModelByID('nonexistent') should return error")
	}
}

func TestModelsForLanguage(t *testing.T) {
	groq := GetProvider("groq")

	// en should include all models
	enModels := ModelsForLanguage(groq, Transcription, "en")
	if len(enModels) != 2 {
		t.Errorf("ModelsForLanguage('en') = %d, want 2", len(enModels))
	}

	// es should include all models (both are multilingual)
	esModels := ModelsForLanguage(groq, Transcription, "es")
	if len(esModels) != 2 {
		t.Errorf("ModelsForLanguage('es') = %d, want 2", len(esModels))
	}

	// auto ("") should include all models
	autoModels := ModelsForLanguage(groq, Transcription, "")
	if len(autoModels) != 2 {
		t.Errorf("ModelsForLanguage('') = %d, want 2 (auto returns all)", len(autoModels))
	}
}

func TestValidateModelLanguage(t *testing.T) {
	// valid language for multilingual model
	err := ValidateModelLanguage("groq", "whisper-large-v3", "es")
	if err != nil {
		t.Errorf("ValidateModelLanguage(whisper-large-v3, 'es') unexpected error: %v", err)
	}

	// valid language for another multilingual model
	err = ValidateModelLanguage("groq", "whisper-large-v3-turbo", "de")
	if err != nil {
		t.Errorf("ValidateModelLanguage(whisper-large-v3-turbo, 'de') unexpected error: %v", err)
	}

	// auto always passes
	err = ValidateModelLanguage("groq", "whisper-large-v3", "")
	if err != nil {
		t.Errorf("ValidateModelLanguage(whisper-large-v3, '') should pass (auto): %v", err)
	}

	// unknown provider
	err = ValidateModelLanguage("nonexistent", "whisper-1", "en")
	if err == nil {
		t.Error("ValidateModelLanguage with unknown provider should return error")
	}

	// unknown model
	err = ValidateModelLanguage("openai", "nonexistent", "en")
	if err == nil {
		t.Error("ValidateModelLanguage with unknown model should return error")
	}
}

func TestValidateModelLanguage_ErrorFormat(t *testing.T) {
	// verify error includes model name, not ID
	// use whisper-cpp base.en model which is English-only
	err := ValidateModelLanguage("whisper-cpp", "base.en", "es")
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
	errMsg := err.Error()

	// should contain model name (from Model.Name)
	if !strings.Contains(errMsg, "Base English") {
		t.Errorf("error should contain model name, got: %s", errMsg)
	}

	// should contain docs URL
	if !strings.Contains(errMsg, "https://github.com/openai/whisper") {
		t.Errorf("error should contain docs URL, got: %s", errMsg)
	}

	// should contain language code
	if !strings.Contains(errMsg, "'es'") {
		t.Errorf("error should contain language code, got: %s", errMsg)
	}

	// should contain supported languages (English-only has just 'en')
	if !strings.Contains(errMsg, "en") {
		t.Errorf("error should contain supported languages, got: %s", errMsg)
	}
}

func TestOpenAIStreamingModels(t *testing.T) {
	// gpt-4o-transcribe supports both batch and streaming
	m, err := GetModel("openai", "gpt-4o-transcribe")
	if err != nil {
		t.Fatalf("GetModel('openai', 'gpt-4o-transcribe') error: %v", err)
	}

	if !m.SupportsBatch {
		t.Error("gpt-4o-transcribe should have SupportsBatch=true")
	}
	if !m.SupportsStreaming {
		t.Error("gpt-4o-transcribe should have SupportsStreaming=true")
	}
	if !m.SupportsBothModes() {
		t.Error("gpt-4o-transcribe should support both modes")
	}

	if m.StreamingAdapter != "openai-realtime" {
		t.Errorf("gpt-4o-transcribe StreamingAdapter=%q, want 'openai-realtime'", m.StreamingAdapter)
	}

	if m.StreamingEndpoint == nil {
		t.Fatal("gpt-4o-transcribe should have StreamingEndpoint set")
	}

	if m.StreamingEndpoint.BaseURL != "wss://api.openai.com" {
		t.Errorf("gpt-4o-transcribe StreamingEndpoint.BaseURL=%q, want 'wss://api.openai.com'", m.StreamingEndpoint.BaseURL)
	}

	// default model should still be whisper-1
	p := GetProvider("openai")
	if p.DefaultModel(Transcription) != "whisper-1" {
		t.Errorf("DefaultModel(Transcription) = %q, want 'whisper-1'", p.DefaultModel(Transcription))
	}
}

func TestElevenLabsProvider(t *testing.T) {
	p := GetProvider("elevenlabs")
	if p == nil {
		t.Fatal("GetProvider('elevenlabs') returned nil")
	}

	models := p.Models()

	// ElevenLabsProvider.Models() returns 3 models
	if len(models) != 3 {
		t.Errorf("ElevenLabsProvider.Models() = %d models, want 3", len(models))
	}

	// Check batch-only models
	scribeV1, err := GetModel("elevenlabs", "scribe_v1")
	if err != nil {
		t.Fatalf("GetModel('elevenlabs', 'scribe_v1') error: %v", err)
	}
	if !scribeV1.SupportsBatch {
		t.Error("scribe_v1 should have SupportsBatch=true")
	}
	if scribeV1.SupportsStreaming {
		t.Error("scribe_v1 should have SupportsStreaming=false")
	}
	if scribeV1.AdapterType != "elevenlabs" {
		t.Errorf("scribe_v1 AdapterType=%q, want 'elevenlabs'", scribeV1.AdapterType)
	}

	scribeV2, err := GetModel("elevenlabs", "scribe_v2")
	if err != nil {
		t.Fatalf("GetModel('elevenlabs', 'scribe_v2') error: %v", err)
	}
	if !scribeV2.SupportsBatch {
		t.Error("scribe_v2 should have SupportsBatch=true")
	}
	if scribeV2.SupportsStreaming {
		t.Error("scribe_v2 should have SupportsStreaming=false")
	}
	if scribeV2.AdapterType != "elevenlabs" {
		t.Errorf("scribe_v2 AdapterType=%q, want 'elevenlabs'", scribeV2.AdapterType)
	}

	// Check streaming-only model
	scribeV2Realtime, err := GetModel("elevenlabs", "scribe_v2_realtime")
	if err != nil {
		t.Fatalf("GetModel('elevenlabs', 'scribe_v2_realtime') error: %v", err)
	}
	if scribeV2Realtime.SupportsBatch {
		t.Error("scribe_v2_realtime should have SupportsBatch=false")
	}
	if !scribeV2Realtime.SupportsStreaming {
		t.Error("scribe_v2_realtime should have SupportsStreaming=true")
	}
	if scribeV2Realtime.AdapterType != "elevenlabs-streaming" {
		t.Errorf("scribe_v2_realtime AdapterType=%q, want 'elevenlabs-streaming'", scribeV2Realtime.AdapterType)
	}

	// All models have explicit SupportedLanguages from docs (subset of our 57)
	for _, m := range models {
		if len(m.SupportedLanguages) != 57 {
			t.Errorf("model %q has %d languages, want 57", m.ID, len(m.SupportedLanguages))
		}
	}
}
