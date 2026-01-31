package transcriber

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/leonardotrapani/hyprvoice/internal/recording"
)

// StreamingTranscriber wraps a StreamingAdapter and implements the Transcriber interface.
// It streams audio chunks to the adapter in real-time and accumulates transcription results.
type StreamingTranscriber struct {
	adapter  StreamingAdapter
	language string

	// accumulated final text
	finalText strings.Builder
	mu        sync.Mutex

	// coordination
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewStreamingTranscriber(adapter StreamingAdapter, language string) *StreamingTranscriber {
	return &StreamingTranscriber{
		adapter:  adapter,
		language: language,
	}
}

func (t *StreamingTranscriber) Start(ctx context.Context, frameCh <-chan recording.AudioFrame) (<-chan error, error) {
	t.ctx, t.cancel = context.WithCancel(ctx)

	if err := t.adapter.Start(t.ctx, t.language); err != nil {
		t.cancel()
		return nil, err
	}

	errCh := make(chan error, 2)

	// goroutine 1: read audio frames and send to adapter
	t.wg.Add(1)
	go t.sendAudio(frameCh, errCh)

	// goroutine 2: read results from adapter and accumulate
	t.wg.Add(1)
	go t.receiveResults(errCh)

	return errCh, nil
}

func (t *StreamingTranscriber) sendAudio(frameCh <-chan recording.AudioFrame, errCh chan<- error) {
	defer t.wg.Done()

	for {
		select {
		case <-t.ctx.Done():
			return
		case frame, ok := <-frameCh:
			if !ok {
				return
			}
			if err := t.adapter.SendChunk(frame.Data); err != nil {
				select {
				case errCh <- err:
				default:
				}
				// don't treat send errors as fatal - adapter may handle reconnection
				log.Printf("streaming transcriber: send error: %v", err)
			}
		}
	}
}

func (t *StreamingTranscriber) receiveResults(errCh chan<- error) {
	defer t.wg.Done()

	resultsCh := t.adapter.Results()
	for {
		select {
		case <-t.ctx.Done():
			return
		case result, ok := <-resultsCh:
			if !ok {
				return
			}
			if result.Error != nil {
				select {
				case errCh <- result.Error:
				default:
				}
				log.Printf("streaming transcriber: result error: %v", result.Error)
				continue
			}
			if result.IsFinal && result.Text != "" {
				t.mu.Lock()
				if t.finalText.Len() > 0 {
					t.finalText.WriteString(" ")
				}
				t.finalText.WriteString(result.Text)
				t.mu.Unlock()
			}
		}
	}
}

func (t *StreamingTranscriber) Stop(ctx context.Context) error {
	if t.cancel != nil {
		t.cancel()
	}

	// wait for goroutines to finish
	t.wg.Wait()

	// close the adapter
	return t.adapter.Close()
}

func (t *StreamingTranscriber) GetFinalTranscription() (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.finalText.String(), nil
}
