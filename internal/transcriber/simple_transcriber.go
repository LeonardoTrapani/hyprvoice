package transcriber

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/leonardotrapani/hyprvoice/internal/recording"
)

// SimpleTranscriber collects all audio and transcribes when stopped
type SimpleTranscriber struct {
	adapter BatchAdapter
	config  Config

	// Audio collection
	audioBuffer []byte
	bufferMu    sync.Mutex

	// Control
	running bool
	wg      sync.WaitGroup

	// Transcription result
	transcriptionMu   sync.RWMutex
	transcriptionText string
}

func NewSimpleTranscriber(config Config, adapter BatchAdapter) *SimpleTranscriber {
	return &SimpleTranscriber{
		adapter: adapter,
		config:  config,
	}
}

func (t *SimpleTranscriber) Start(ctx context.Context, frameCh <-chan recording.AudioFrame) (<-chan error, error) {
	if t.running {
		return nil, fmt.Errorf("transcriber already running")
	}

	t.running = true

	errCh := make(chan error, 1)

	t.wg.Add(1)
	go t.collectAudio(ctx, frameCh, errCh)

	return errCh, nil
}

func (t *SimpleTranscriber) Stop(ctx context.Context) error {
	if !t.running {
		return nil
	}

	t.wg.Wait()
	t.running = false

	// Transcribe all collected audio using the passed context
	return t.transcribeAll(ctx)
}

func (t *SimpleTranscriber) GetFinalTranscription() (string, error) {
	t.transcriptionMu.RLock()
	defer t.transcriptionMu.RUnlock()
	return t.transcriptionText, nil
}

func (t *SimpleTranscriber) collectAudio(ctx context.Context, frameCh <-chan recording.AudioFrame, errCh chan<- error) {
	defer func() {
		close(errCh)
		t.wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("transcriber: stopping audio collection")
			return

		case frame, ok := <-frameCh:
			if !ok {
				log.Printf("transcriber: audio channel closed")
				return
			}

			t.bufferMu.Lock()
			t.audioBuffer = append(t.audioBuffer, frame.Data...)
			t.bufferMu.Unlock()
		}
	}
}

func (t *SimpleTranscriber) transcribeAll(ctx context.Context) error {
	t.bufferMu.Lock()
	audioData := make([]byte, len(t.audioBuffer))
	copy(audioData, t.audioBuffer)
	t.bufferMu.Unlock()

	if len(audioData) == 0 {
		log.Printf("transcriber: no audio data to transcribe")
		return nil
	}

	log.Printf("transcriber: transcribing %d bytes of audio", len(audioData))

	// Use the context passed from the pipeline for proper cancellation chain
	text, err := t.adapter.Transcribe(ctx, audioData)
	if err != nil {
		log.Printf("transcriber: transcription failed: %v", err)
		return fmt.Errorf("transcription failed: %w", err)
	}

	log.Printf("transcriber: transcription completed: %q", text)

	t.transcriptionMu.Lock()
	t.transcriptionText = text
	t.transcriptionMu.Unlock()

	return nil
}
