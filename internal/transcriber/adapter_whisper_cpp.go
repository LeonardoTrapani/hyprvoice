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

// WhisperCppAdapter implements BatchAdapter for local whisper-cpp transcription
type WhisperCppAdapter struct {
	modelPath string
	language  string
	threads   int
}

// NewWhisperCppAdapter creates a new whisper-cpp adapter
// modelPath: full path to the model file (e.g., ~/.local/share/hyprvoice/models/whisper/ggml-base.en.bin)
// lang: whisper-cpp language code
// threads: number of CPU threads (0 for auto)
func NewWhisperCppAdapter(modelPath, lang string, threads int) *WhisperCppAdapter {
	return &WhisperCppAdapter{
		modelPath: modelPath,
		language:  lang,
		threads:   threads,
	}
}

func (a *WhisperCppAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if len(audioData) == 0 {
		return "", nil
	}

	// check model file exists
	if _, err := os.Stat(a.modelPath); os.IsNotExist(err) {
		return "", fmt.Errorf("model file not found: %s", a.modelPath)
	}

	// check whisper-cli exists
	whisperPath, err := exec.LookPath("whisper-cli")
	if err != nil {
		return "", fmt.Errorf("whisper-cli not found: install whisper.cpp first")
	}

	// convert raw PCM to WAV
	wavData, err := convertToWAV(audioData)
	if err != nil {
		return "", fmt.Errorf("convert to WAV: %w", err)
	}

	// write to temp file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("hyprvoice-%d.wav", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, wavData, 0600); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}
	defer os.Remove(tmpFile)

	// use whisper-cpp auto if unspecified
	lang := a.language
	if lang == "" {
		lang = "auto"
	}

	// build command args
	args := []string{
		"-m", a.modelPath,
		"-l", lang,
		"-nt", // no timestamps
		"-np", // no progress
		"-f", tmpFile,
	}

	// add threads if specified
	if a.threads > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", a.threads))
	}

	// execute whisper-cli
	cmd := exec.CommandContext(ctx, whisperPath, args...)
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
		log.Printf("whisper-cpp: command failed after %v: %v\nstderr: %s", duration, err, stderr.String())
		return "", fmt.Errorf("whisper-cli failed: %w", err)
	}

	// parse output - whisper-cli outputs transcription text directly (with -nt flag)
	text := strings.TrimSpace(stdout.String())

	log.Printf("whisper-cpp: transcribed %d bytes in %v: %q", len(audioData), duration, text)
	return text, nil
}
