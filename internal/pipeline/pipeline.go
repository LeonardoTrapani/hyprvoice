package pipeline

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/recording"
)

type Status string
type Action string

const (
	Idle         Status = "idle"
	Recording    Status = "recording"
	Transcribing Status = "transcribing"
	Injecting    Status = "injecting"
)

const (
	Inject Action = "inject"
)

type Pipeline interface {
	Run(ctx context.Context)
	Stop()
	Status() Status
	Actions() chan<- Action
}

type pipeline struct {
	status   Status
	actionCh chan Action
	wg       sync.WaitGroup
	cancel   context.CancelFunc
}

func New() Pipeline {
	return &pipeline{
		actionCh: make(chan Action, 1),
	}
}

func (p *pipeline) Status() Status {
	return p.status
}

func (p *pipeline) Actions() chan<- Action {
	return p.actionCh
}

func (p *pipeline) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

func (p *pipeline) Run(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	p.cancel = cancel
	p.wg.Add(1)
	go p.run(runCtx)
}

func (p *pipeline) run(ctx context.Context) {
	defer func() {
		p.wg.Done()
	}()

	log.Printf("Pipeline: Starting recording")
	p.status = Recording

	recorder := recording.NewDefaultRecorder()
	frameCh, errCh, err := recorder.Start(ctx)
	if err != nil {
		log.Printf("Pipeline: Recording error: %v", err)
		p.status = Idle
		return
	}
	defer recorder.Stop()

	frameCount := 0
	totalBytes := 0

	for {
		select {
		case frame := <-frameCh:
			frameCount++
			totalBytes += len(frame.Data)
			log.Printf("Pipeline: Received frame #%d - Size: %d bytes, Timestamp: %v, Total bytes so far: %d",
				frameCount, len(frame.Data), frame.Timestamp.Format("15:04:05.000"), totalBytes)

			p.status = Transcribing

		case err := <-errCh:
			if err != nil {
				log.Printf("Pipeline: Recording error: %v", err)
				p.status = Idle
				return
			}

		case action := <-p.actionCh:
			log.Printf("Pipeline: Received action: %v", action)
			switch action {
			case Inject:
				log.Printf("Pipeline: Inject action received, stopping recording (TODO: stop transcribing)")

				if err := recorder.Stop(); err != nil {
					log.Printf("Pipeline: Error stopping recorder: %v", err)
				}
				p.status = Injecting
				return
			}

		case <-ctx.Done():
			log.Printf("Pipeline: Context cancelled, stopping")
			return
		}
	}
}
