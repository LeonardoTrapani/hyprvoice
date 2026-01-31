package provider

import (
	"slices"
	"testing"
)

func TestProviderInterface(t *testing.T) {
	providers := []struct {
		name              string
		hasTranscription  bool
		hasLLM            bool
		defaultTransModel string
		defaultLLMModel   string
	}{
		{"openai", true, true, "whisper-1", "gpt-4o-mini"},
		{"groq", true, true, "whisper-large-v3-turbo", "llama-3.3-70b-versatile"},
		{"mistral", true, false, "voxtral-mini-latest", ""},
		{"elevenlabs", true, false, "scribe_v1", ""},
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

			if p.SupportsTranscription() != tc.hasTranscription {
				t.Errorf("SupportsTranscription() = %v, want %v", p.SupportsTranscription(), tc.hasTranscription)
			}

			if p.SupportsLLM() != tc.hasLLM {
				t.Errorf("SupportsLLM() = %v, want %v", p.SupportsLLM(), tc.hasLLM)
			}

			if p.DefaultTranscriptionModel() != tc.defaultTransModel {
				t.Errorf("DefaultTranscriptionModel() = %q, want %q", p.DefaultTranscriptionModel(), tc.defaultTransModel)
			}

			if p.DefaultLLMModel() != tc.defaultLLMModel {
				t.Errorf("DefaultLLMModel() = %q, want %q", p.DefaultLLMModel(), tc.defaultLLMModel)
			}

			if !p.RequiresAPIKey() {
				t.Error("RequiresAPIKey() should be true for all providers")
			}

			if tc.hasTranscription && len(p.TranscriptionModels()) == 0 {
				t.Error("TranscriptionModels() should not be empty for transcription provider")
			}

			if tc.hasLLM && len(p.LLMModels()) == 0 {
				t.Error("LLMModels() should not be empty for LLM provider")
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
