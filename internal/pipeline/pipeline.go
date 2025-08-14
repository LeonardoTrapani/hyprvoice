package pipeline

import (
	"context"
	"log"
	"sync"
	"time"
)

type Status string

const (
	Idle         Status = "idle"
	Recording    Status = "recording"
	Transcribing Status = "transcribing"
	Injecting    Status = "injecting"
)

// Action represents commands sent to the pipeline to drive state
// transitions or trigger work. Keeping this separate from Status
// preserves directionality: callers send Actions, pipeline emits Statuses.
type Action int

const (
	Inject Action = iota
)

type Pipeline interface {
	Run(ctx context.Context) (<-chan Status, chan<- Action)
	Stop()
}

type pipeline struct {
	actionCh chan Action
	wg       sync.WaitGroup
	cancel   context.CancelFunc
}

func New() Pipeline {
	return &pipeline{
		actionCh: make(chan Action, 1),
	}
}

func (p *pipeline) Run(ctx context.Context) (<-chan Status, chan<- Action) {
	statusCh := make(chan Status, 1)
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	p.cancel = cancel
	p.wg.Add(1)
	go p.run(runCtx, statusCh)
	return statusCh, p.actionCh
}

func (p *pipeline) run(ctx context.Context, statusCh chan<- Status) {
	defer func() {
		// Ensure status channel is closed and wg is decremented
		close(statusCh)
		p.wg.Done()
	}()

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

	// TODO: when integrating the recorder, ensure its context is cancelled here on exit paths
	// Wait for an action or timeout
	select {
	case action := <-p.actionCh:
		switch action {
		case Inject:
			log.Printf("Pipeline: Injection started")
			statusCh <- Injecting

			// Injection work
			select {
			case <-time.After(1 * time.Second):
				log.Printf("Pipeline: Injection complete")
				statusCh <- Idle
			case <-ctx.Done():
				log.Printf("Pipeline: Stopped during injection")
				return
			}
		default:
			// Unknown action: ignore for now
		}

	case <-time.After(10 * time.Second): // use context timeout
		log.Printf("Pipeline: Auto-timeout, completing")
		statusCh <- Idle // Instead of Completed

	case <-ctx.Done():
		log.Printf("Pipeline: Stopped during transcription wait")
		return
	}
}

func (p *pipeline) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}
