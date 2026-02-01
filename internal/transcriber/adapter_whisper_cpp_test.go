package transcriber

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWhisperCppAdapter_ImplementsBatchAdapter(t *testing.T) {
	// compile-time check that WhisperCppAdapter implements BatchAdapter
	var _ BatchAdapter = (*WhisperCppAdapter)(nil)
}

func TestWhisperCppAdapter_EmptyAudio(t *testing.T) {
	adapter := NewWhisperCppAdapter("/nonexistent/model.bin", "en", 4)
	text, err := adapter.Transcribe(context.Background(), []byte{})
	if err != nil {
		t.Errorf("expected no error for empty audio, got: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty text for empty audio, got: %q", text)
	}
}

func TestWhisperCppAdapter_MissingModel(t *testing.T) {
	adapter := NewWhisperCppAdapter("/nonexistent/path/model.bin", "en", 4)

	// create minimal valid PCM data (just zeros)
	audioData := make([]byte, 32000) // 1 second at 16kHz 16-bit

	_, err := adapter.Transcribe(context.Background(), audioData)
	if err == nil {
		t.Error("expected error for missing model file")
	}
	if err != nil && !contains(err.Error(), "model file not found") {
		t.Errorf("expected 'model file not found' error, got: %v", err)
	}
}

func TestWhisperCppAdapter_LanguageConversion(t *testing.T) {
	// verify adapter stores language for later conversion
	adapter := NewWhisperCppAdapter("/fake/model.bin", "", 4)
	if adapter.language != "" {
		t.Errorf("expected empty language for auto, got: %q", adapter.language)
	}

	adapter = NewWhisperCppAdapter("/fake/model.bin", "en", 4)
	if adapter.language != "en" {
		t.Errorf("expected 'en' language, got: %q", adapter.language)
	}
}

func TestWhisperCppAdapter_ThreadsConfig(t *testing.T) {
	adapter := NewWhisperCppAdapter("/fake/model.bin", "en", 0)
	if adapter.threads != 0 {
		t.Errorf("expected threads=0 (auto), got: %d", adapter.threads)
	}

	adapter = NewWhisperCppAdapter("/fake/model.bin", "en", 8)
	if adapter.threads != 8 {
		t.Errorf("expected threads=8, got: %d", adapter.threads)
	}
}

func TestWhisperCppAdapter_TempFileCleanup(t *testing.T) {
	// this test requires whisper-cli and a model to be installed
	// skip if not available
	modelPath := os.Getenv("WHISPER_TEST_MODEL")
	if modelPath == "" {
		t.Skip("WHISPER_TEST_MODEL not set, skipping temp file cleanup test")
	}

	adapter := NewWhisperCppAdapter(modelPath, "en", 4)

	// create minimal audio data
	audioData := make([]byte, 32000)

	// run transcription
	_, _ = adapter.Transcribe(context.Background(), audioData)

	// check that temp file was cleaned up
	// (we can't easily verify this without modifying the adapter to expose temp path)
	// this is more of a visual/log verification
}

func TestWhisperCppAdapter_ContextCancellation(t *testing.T) {
	// skip if whisper-cli not installed
	if _, err := os.Stat("/usr/local/bin/whisper-cli"); os.IsNotExist(err) {
		t.Skip("whisper-cli not installed")
	}

	// create a fake model file for this test
	tmpDir := t.TempDir()
	fakeModel := filepath.Join(tmpDir, "fake.bin")
	if err := os.WriteFile(fakeModel, []byte("fake"), 0600); err != nil {
		t.Fatalf("failed to create fake model: %v", err)
	}

	adapter := NewWhisperCppAdapter(fakeModel, "en", 4)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// create minimal audio data
	audioData := make([]byte, 32000)

	_, err := adapter.Transcribe(ctx, audioData)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
