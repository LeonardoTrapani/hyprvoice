package transcriber

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ParakeetAdapter implements BatchAdapter for local NVIDIA Parakeet transcription
// via the NeMo-Speech.cpp runtime (nemo-speech CLI).
type ParakeetAdapter struct {
	modelPath string
	device    string
}

// NewParakeetAdapter creates a new parakeet adapter
// modelPath: full path to the model file (e.g., ~/.local/share/hyprvoice/models/parakeet/parakeet-tdt-0.6b-v3.q8_0.gguf)
// device: inference device ("" for auto, "cpu", "cuda:0", "vulkan:0")
func NewParakeetAdapter(modelPath, device string) *ParakeetAdapter {
	return &ParakeetAdapter{
		modelPath: modelPath,
		device:    device,
	}
}

func (a *ParakeetAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if len(audioData) == 0 {
		return "", nil
	}

	// check model file exists
	if _, err := os.Stat(a.modelPath); os.IsNotExist(err) {
		return "", fmt.Errorf("model file not found: %s (run 'hyprvoice model download parakeet-tdt-0.6b-v3')", a.modelPath)
	}

	// check nemo-speech exists
	speechPath, err := exec.LookPath("nemo-speech")
	if err != nil {
		return "", fmt.Errorf("nemo-speech not found: install NeMo-Speech.cpp (see https://github.com/NVIDIA/NeMo-Speech.cpp)")
	}

	// convert raw PCM to WAV
	wavData, err := convertToWAV(audioData)
	if err != nil {
		return "", fmt.Errorf("convert to WAV: %w", err)
	}

	// write to temp file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("hyprvoice-parakeet-%d.wav", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, wavData, 0600); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}
	defer os.Remove(tmpFile)

	// build command args
	args := []string{"transcribe", tmpFile, "--model", a.modelPath, "--quiet"}
	if a.device != "" {
		args = append(args, "--device", a.device)
	}

	// execute nemo-speech
	cmd := exec.CommandContext(ctx, speechPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err = cmd.Run()
	duration := time.Since(start)

	if err != nil {
		// check if context was cancelled
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		log.Printf("parakeet: command failed after %v: %v\nstderr: %s", duration, err, stderr.String())
		return "", fmt.Errorf("nemo-speech failed: %w", err)
	}

	// parse output - nemo-speech writes plain transcription text to stdout
	text := strings.TrimSpace(stdout.String())

	log.Printf("parakeet: transcribed %d bytes in %v: %q", len(audioData), duration, text)
	return text, nil
}
