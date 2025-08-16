package pipeline

import (
	"context"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/recording"
	"github.com/leonardotrapani/hyprvoice/internal/transcriber"
)

type Status string
type Action string

type PipelineError struct {
	Title   string
	Message string
	Err     error
}

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
	GetErrorCh() <-chan PipelineError
}

type pipeline struct {
	status   Status
	actionCh chan Action
	errorCh  chan PipelineError

	mu       sync.RWMutex
	wg       sync.WaitGroup
	cancel   context.CancelFunc
	stopOnce sync.Once

	running int32
}

func New() Pipeline {
	return &pipeline{
		actionCh: make(chan Action, 1),
		errorCh:  make(chan PipelineError, 10),
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

func (p *pipeline) GetErrorCh() <-chan PipelineError {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.errorCh
}

func (p *pipeline) sendError(title, message string, err error) {
	pipelineErr := PipelineError{
		Title:   title,
		Message: message,
		Err:     err,
	}

	select {
	case p.errorCh <- pipelineErr:
	default:
		log.Printf("Pipeline: Error channel full, dropping error: %s", message)
	}
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
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		log.Printf("Pipeline: Already running, ignoring Run() call")
		return
	}

	runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	p.setCancel(cancel)

	p.wg.Add(1)
	go p.run(runCtx)
}

func (p *pipeline) run(ctx context.Context) {
	defer func() {
		atomic.StoreInt32(&p.running, 0)
		p.setStatus(Idle)
		p.wg.Done()
	}()

	log.Printf("Pipeline: Starting recording")
	p.setStatus(Recording)

	recorder := recording.NewDefaultRecorder()
	frameCh, errCh, err := recorder.Start(ctx)
	if err != nil {
		log.Printf("Pipeline: Recording error: %v", err)
		p.sendError("Recording Error", "Failed to start recording", err)
		return
	}

	defer func() {
		if stopErr := recorder.Stop(); stopErr != nil {
			log.Printf("Pipeline: Error stopping recorder: %v", stopErr)
			p.sendError("Recording Error", "Failed to stop recorder cleanly", stopErr)
		}
	}()

	config := transcriber.DefaultConfig()
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}

	t, err := transcriber.NewTranscriber(config)
	if err != nil {
		log.Printf("Pipeline: Failed to create transcriber: %v", err)
		p.sendError("Transcription Error", "Failed to create transcriber", err)
		return
	}

	log.Printf("Pipeline: Starting transcriber")
	p.setStatus(Transcribing)

	tErrCh, err := t.Start(ctx, frameCh)
	if err != nil {
		log.Printf("Pipeline: Transcriber error: %v", err)
		p.sendError("Transcription Error", "Failed to start transcriber", err)
		return
	}

	defer func() {
		if stopErr := t.Stop(); stopErr != nil {
			log.Printf("Pipeline: Error stopping transcriber: %v", stopErr)
			p.sendError("Transcription Error", "Failed to stop transcriber cleanly", stopErr)
		}
	}()

	frameCount := 0
	totalBytes := 0

	for {
		select {
		case frame := <-frameCh:
			frameCount++
			totalBytes += len(frame.Data)
			log.Printf("Pipeline: Received frame #%d - Size: %d bytes, Timestamp: %v, Total bytes so far: %d",
				frameCount, len(frame.Data), frame.Timestamp.Format("15:04:05.000"), totalBytes)

		case err := <-tErrCh:
			if err != nil {
				log.Printf("Pipeline: Transcription error: %v", err)
				p.sendError("Transcription Error", "Transcription processing error", err)
				return
			}

		case err := <-errCh:
			if err != nil {
				log.Printf("Pipeline: Recording error: %v", err)
				p.sendError("Recording Error", "Recording stream error", err)
				return
			}

		case action := <-p.actionCh:
			log.Printf("Pipeline: Received action: %v", action)
			switch action {
			case Inject:
				if p.status != Transcribing {
					log.Printf("Pipeline: Inject action received, but not in transcribing state, ignoring")
					continue
				}

				log.Printf("Pipeline: Inject action received, stopping recording and getting transcription")

				if err := recorder.Stop(); err != nil {
					log.Printf("Pipeline: Error stopping recorder: %v", err)
					p.sendError("Recording Error", "Failed to stop recorder during injection", err)
				}

				p.setStatus(Injecting)

				transcriptionText, err := t.GetTranscription()
				if err != nil {
					log.Printf("Pipeline: Error getting transcription: %v", err)
					p.sendError("Transcription Error", "Failed to retrieve transcription", err)
				} else {
					log.Printf("Pipeline: Transcription text: %s", transcriptionText)
				}

				log.Printf("Pipeline: Simulating injection work")
				time.Sleep(10 * time.Millisecond)
				log.Printf("Pipeline: Injection work done, returning to idle")
				return
			}

		case <-ctx.Done():
			log.Printf("Pipeline: Context cancelled, stopping")
			return
		}
	}
}
