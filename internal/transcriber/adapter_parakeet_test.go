package transcriber

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParakeetAdapter_ImplementsBatchAdapter(t *testing.T) {
	// compile-time check that ParakeetAdapter implements BatchAdapter
	var _ BatchAdapter = (*ParakeetAdapter)(nil)
}

func TestParakeetAdapter_EmptyAudio(t *testing.T) {
	adapter := NewParakeetAdapter("/nonexistent/model.bin", "")
	text, err := adapter.Transcribe(context.Background(), []byte{})
	if err != nil {
		t.Errorf("expected no error for empty audio, got: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty text for empty audio, got: %q", text)
	}
}

func TestParakeetAdapter_MissingModel(t *testing.T) {
	adapter := NewParakeetAdapter("/nonexistent/path/model.bin", "")

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

func TestParakeetAdapter_MissingCli(t *testing.T) {
	if _, err := exec.LookPath("nemo-speech"); err == nil {
		t.Skip("nemo-speech is installed")
	}

	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "model.gguf")
	if err := os.WriteFile(modelPath, []byte("fake"), 0600); err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	adapter := NewParakeetAdapter(modelPath, "")
	audioData := make([]byte, 32000)

	_, err := adapter.Transcribe(context.Background(), audioData)
	if err == nil {
		t.Error("expected error for missing nemo-speech")
	}
	if err != nil && !contains(err.Error(), "nemo-speech not found") {
		t.Errorf("expected 'nemo-speech not found' error, got: %v", err)
	}
}

func TestParakeetAdapter_DeviceConfig(t *testing.T) {
	adapter := NewParakeetAdapter("/fake/model.bin", "")
	if adapter.device != "" {
		t.Errorf("expected empty device (auto), got: %q", adapter.device)
	}

	adapter = NewParakeetAdapter("/fake/model.bin", "cpu")
	if adapter.device != "cpu" {
		t.Errorf("expected 'cpu' device, got: %q", adapter.device)
	}
}
