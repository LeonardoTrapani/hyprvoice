package transcriber

import (
	"context"
	"testing"
)

func TestNewElevenLabsAdapter(t *testing.T) {
	config := Config{
		Provider: "elevenlabs",
		APIKey:   "test-api-key",
		Language: "en",
		Model:    "scribe_v1",
	}

	adapter := NewElevenLabsAdapter(config)

	if adapter == nil {
		t.Fatalf("NewElevenLabsAdapter() returned nil")
	}

	if adapter.config.APIKey != "test-api-key" {
		t.Errorf("APIKey not set correctly, got: %s", adapter.config.APIKey)
	}

	if adapter.config.Model != "scribe_v1" {
		t.Errorf("Model not set correctly, got: %s", adapter.config.Model)
	}
}

func TestElevenLabsAdapter_Transcribe_EmptyAudio(t *testing.T) {
	config := Config{
		Provider: "elevenlabs",
		APIKey:   "test-key",
		Model:    "scribe_v1",
	}

	adapter := NewElevenLabsAdapter(config)
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
	// This test will require mocking the HTTP client
	// For now, we test the structure exists
	config := Config{
		Provider: "elevenlabs",
		APIKey:   "test-key",
		Language: "en",
		Model:    "scribe_v1",
	}

	adapter := NewElevenLabsAdapter(config)

	if adapter == nil {
		t.Fatal("NewElevenLabsAdapter() returned nil")
	}

	// Test that adapter has a client
	if adapter.client == nil {
		t.Error("adapter.client is nil")
	}
}
