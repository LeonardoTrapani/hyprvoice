package pipeline

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/recording"
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

	recorder := recording.NewDefaultRecorder()
	frameCh, errCh, err := recorder.Start(ctx)
	if err != nil {
		log.Printf("Pipeline: Recording error: %v", err)
		statusCh <- Idle
		return
	}
	defer recorder.Stop()

	// Track statistics for logging
	frameCount := 0
	totalBytes := 0
	startTime := time.Now()

	// Main event loop
	for {
		select {
		case frame, ok := <-frameCh:
			if !ok {
				// Frame channel closed, recording ended
				log.Printf("Pipeline: Recording ended. Total frames: %d, Total bytes: %d, Duration: %v",
					frameCount, totalBytes, time.Since(startTime))
				statusCh <- Transcribing

				return
			}
			// Log frame details
			frameCount++
			totalBytes += len(frame.Data)
			log.Printf("Pipeline: Received frame #%d - Size: %d bytes, Timestamp: %v, Total bytes so far: %d",
				frameCount, len(frame.Data), frame.Timestamp.Format("15:04:05.000"), totalBytes)
			// TODO: pass this channel to the transcriber

		case err := <-errCh:
			if err != nil {
				log.Printf("Pipeline: Recording error: %v", err)
				statusCh <- Idle
				return
			}

		case action := <-p.actionCh:
			log.Printf("Pipeline: Received action: %v", action)
			if action == Inject {
				log.Printf("Pipeline: Inject action received, stopping recording")

				if err := recorder.Stop(); err != nil {
					log.Printf("Pipeline: Error stopping recorder: %v", err)
				}
				statusCh <- Injecting
				return
			}

		case <-ctx.Done():
			log.Printf("Pipeline: Context cancelled, stopping")
			return
		}
	}
}

func (p *pipeline) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}
