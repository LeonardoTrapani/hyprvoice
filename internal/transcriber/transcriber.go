package transcriber

import (
	"context"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/recording"
)

type TranscriptionResult struct {
	Text      string
	Timestamp time.Time
	IsFinal   bool
}

type Transcriber interface {
	Start(ctx context.Context, frameCh <-chan recording.AudioFrame) (<-chan error, error)
	Stop() error
	GetTranscription() (string, error)
}

type Config struct {
	Provider   string
	APIKey     string
	Language   string
	ChunkSize  int
	BufferTime time.Duration
	Model      string
}

func DefaultConfig() Config {
	return Config{
		Provider:   "openai",
		Language:   "en",
		ChunkSize:  16384,
		BufferTime: 2 * time.Second,
		Model:      "whisper-1",
	}
}
