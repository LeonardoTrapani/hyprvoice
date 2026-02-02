package transcriber

import (
	"context"
	"testing"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

func TestNewElevenLabsAdapter(t *testing.T) {
	endpoint := &provider.EndpointConfig{
		BaseURL: "https://api.elevenlabs.io",
		Path:    "/v1/speech-to-text",
	}

	adapter := NewElevenLabsAdapter(endpoint, "test-api-key", "scribe_v1", "en", nil)

	if adapter == nil {
		t.Fatalf("NewElevenLabsAdapter() returned nil")
	}

	if adapter.apiKey != "test-api-key" {
		t.Errorf("APIKey not set correctly, got: %s", adapter.apiKey)
	}

	if adapter.model != "scribe_v1" {
		t.Errorf("Model not set correctly, got: %s", adapter.model)
	}

	if adapter.language != "en" {
		t.Errorf("Language not set correctly, got: %s", adapter.language)
	}

	if adapter.endpoint.BaseURL != "https://api.elevenlabs.io" {
		t.Errorf("Endpoint BaseURL not set correctly, got: %s", adapter.endpoint.BaseURL)
	}

	if adapter.endpoint.Path != "/v1/speech-to-text" {
		t.Errorf("Endpoint Path not set correctly, got: %s", adapter.endpoint.Path)
	}
}

func TestElevenLabsAdapter_Transcribe_EmptyAudio(t *testing.T) {
	endpoint := &provider.EndpointConfig{
		BaseURL: "https://api.elevenlabs.io",
		Path:    "/v1/speech-to-text",
	}

	adapter := NewElevenLabsAdapter(endpoint, "test-key", "scribe_v1", "", nil)
	ctx := context.Background()

	result, err := adapter.Transcribe(ctx, []byte{})

	if err != nil {
		t.Errorf("Transcribe() with empty audio should not error, got: %v", err)
	}

	if result != "" {
		t.Errorf("Transcribe() with empty audio should return empty string, got: %s", result)
	}
}

func TestElevenLabsAdapter_Transcribe_ValidAudio(t *testing.T) {
	endpoint := &provider.EndpointConfig{
		BaseURL: "https://api.elevenlabs.io",
		Path:    "/v1/speech-to-text",
	}

	adapter := NewElevenLabsAdapter(endpoint, "test-key", "scribe_v1", "en", nil)

	if adapter == nil {
		t.Fatal("NewElevenLabsAdapter() returned nil")
	}

	// test that adapter has a client
	if adapter.client == nil {
		t.Error("adapter.client is nil")
	}
}
