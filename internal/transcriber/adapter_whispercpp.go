package transcriber

import (
	"context"
	"fmt"
)

// WhisperCppAdapter implements TranscriptionAdapter for local whisper.cpp
type WhisperCppAdapter struct {
	config    Config
	modelPath string
}

func NewWhisperCppAdapter(config Config) *WhisperCppAdapter {
	return &WhisperCppAdapter{
		config: config,
		// TODO: Configure model path based on config
		modelPath: "./models/ggml-base.en.bin",
	}
}

func (a *WhisperCppAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	// TODO: Implement whisper.cpp transcription
	// This will involve:
	// 1. Writing audio data to a temporary file or pipe
	// 2. Calling whisper.cpp binary with appropriate flags
	// 3. Reading the transcription result
	// 4. Cleaning up temporary files

	return "", fmt.Errorf("whisper.cpp adapter not implemented yet")
}
