package pipeline

import (
	"context"
	"log"
	"time"
)

type Status string

const (
	Idle         Status = "idle"
	Recording    Status = "recording"
	Transcribing Status = "transcribing"
	Injecting    Status = "injecting"
)

type Pipeline interface {
	Run(ctx context.Context) <-chan Status
	Inject()
}

type pipeline struct {
	injectCh chan struct{}
}

func New() Pipeline {
	return &pipeline{
		injectCh: make(chan struct{}, 1),
	}
}

func (p *pipeline) Run(ctx context.Context) <-chan Status {
	statusCh := make(chan Status, 1)
	go p.run(ctx, statusCh)
	return statusCh
}

func (p *pipeline) Inject() {
	select {
	case p.injectCh <- struct{}{}:
	default:
	}
}

func (p *pipeline) run(ctx context.Context, statusCh chan<- Status) {
	defer close(statusCh)

	// Start recording
	log.Printf("Pipeline: Starting recording")
	statusCh <- Recording

	// Recording phase
	select {
	case <-time.After(2 * time.Second):
		log.Printf("Pipeline: TODO start recording, and on first chunk set transcribing and start streaming with Whisper")
		statusCh <- Transcribing
	case <-ctx.Done():
		log.Printf("Pipeline: Stopped during recording")
		return
	}

	// Wait for injection or timeout
	select {
	case <-p.injectCh:
		log.Printf("Pipeline: Injection started")
		statusCh <- Injecting

		// Injection work
		select {
		case <-time.After(1 * time.Second):
			log.Printf("Pipeline: Injection complete")
			statusCh <- Idle // Instead of Completed
		case <-ctx.Done():
			log.Printf("Pipeline: Stopped during injection")
			return
		}

	case <-time.After(10 * time.Second): // use context timeout
		log.Printf("Pipeline: Auto-timeout, completing")
		statusCh <- Idle // Instead of Completed

	case <-ctx.Done():
		log.Printf("Pipeline: Stopped during transcription wait")
		return
	}
}
