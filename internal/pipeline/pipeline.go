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
	GetActionCh() chan<- Action
}

type pipeline struct {
	status   Status
	actionCh chan Action

	mu sync.RWMutex
	wg sync.WaitGroup

	cancel context.CancelFunc

	stopOnce sync.Once
	running  bool
}

func New() Pipeline {
	return &pipeline{
		actionCh: make(chan Action, 1),
	}
}

func (p *pipeline) Status() Status {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

func (p *pipeline) setStatus(status Status) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = status
}

func (p *pipeline) setCancel(cancel context.CancelFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cancel = cancel
}

func (p *pipeline) getCancel() context.CancelFunc {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cancel
}

func (p *pipeline) GetActionCh() chan<- Action {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.actionCh
}

func (p *pipeline) Stop() {
	p.stopOnce.Do(func() {
		cancel := p.getCancel()
		if cancel != nil {
			cancel()
		}
	})
	p.wg.Wait()
}

func (p *pipeline) Run(ctx context.Context) {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		log.Printf("Pipeline: Already running, ignoring Run() call")
		return
	}
	p.running = true
	p.mu.Unlock()

	runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	p.setCancel(cancel)

	p.wg.Add(1)
	go p.run(runCtx)
}

func (p *pipeline) run(ctx context.Context) {
	defer func() {
		log.Printf("Pipeline: defered run")
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		p.wg.Done()
	}()

	log.Printf("Pipeline: Starting recording")
	p.setStatus(Recording)

	recorder := recording.NewDefaultRecorder()
	frameCh, errCh, err := recorder.Start(ctx)
	if err != nil {
		log.Printf("Pipeline: Recording error: %v", err)
		p.setStatus(Idle)
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

			p.setStatus(Transcribing)

		case err := <-errCh:
			if err != nil {
				log.Printf("Pipeline: Recording error: %v", err)
				p.setStatus(Idle)
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
				p.setStatus(Injecting)

				// Simulate injection work then return to idle
				log.Printf("Pipeline: Simulating injection work")
				time.Sleep(10 * time.Millisecond)
				log.Printf("Pipeline: Injection work done, returning to idle")
				p.setStatus(Idle)
				return
			}

		case <-ctx.Done():
			log.Printf("Pipeline: Context cancelled, stopping")
			return
		}
	}
}
