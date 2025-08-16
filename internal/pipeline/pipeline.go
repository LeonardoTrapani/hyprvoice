package pipeline

import (
	"context"
	"log"
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

	running atomic.Bool
}

func New() Pipeline {
	return &pipeline{
		actionCh: make(chan Action, 1),
		errorCh:  make(chan PipelineError, 10),
	}
}
func (p *pipeline) Run(ctx context.Context) {
	if !p.running.CompareAndSwap(false, true) {
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
		p.running.Store(false)
		p.setStatus(Idle)
		p.wg.Done()
	}()

	log.Printf("Pipeline: Starting recording")
	p.setStatus(Recording)

	recorder := recording.NewDefaultRecorder()
	frameCh, rErrCh, err := recorder.Start(ctx)

	if err != nil {
		log.Printf("Pipeline: Recording error: %v", err)
		p.sendError("Recording Error", "Failed to start recording", err)
		return
	}

	defer recorder.Stop()

	t, err := transcriber.NewDefaultTranscriber()
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
		if stopErr := t.Stop(ctx); stopErr != nil {
			log.Printf("Pipeline: Error stopping transcriber: %v", stopErr)
			p.sendError("Transcription Error", "Failed to stop transcriber cleanly", stopErr)
		}
	}()

	for {
		select {
		case <-frameCh:

		case action := <-p.actionCh:
			switch action {
			case Inject:
				p.handleInjectAction(ctx, recorder, t)
				return
			}

		case err := <-tErrCh:
			p.handleTranscriberError(err)
			return

		case err := <-rErrCh:
			p.handleRecordingError(err)
			return

		case <-ctx.Done():
			return
		}
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

func (p *pipeline) handleTranscriberError(err error) {
	p.sendError("Transcription Error", "Transcription processing error", err)
}

func (p *pipeline) handleRecordingError(err error) {
	p.sendError("Recording Error", "Recording stream error", err)
}

func (p *pipeline) handleInjectAction(ctx context.Context, recorder *recording.Recorder, t transcriber.Transcriber) {
	status := p.Status()

	if status != Transcribing {
		log.Printf("Pipeline: Inject action received, but not in transcribing state, ignoring")
		return
	}

	log.Printf("Pipeline: Inject action received, stopping recording and finalizing transcription")
	p.setStatus(Injecting)

	recorder.Stop()

	if err := t.Stop(ctx); err != nil {
		p.sendError("Transcription Error", "Failed to stop transcriber during injection", err)
	}

	transcriptionText, err := t.GetFinalTranscription()
	if err != nil {
		p.sendError("Transcription Error", "Failed to retrieve transcription", err)
		return
	}
	log.Printf("Pipeline: Final transcription text: %s", transcriptionText)

	log.Printf("Pipeline: Simulating injection work")
	time.Sleep(10 * time.Millisecond)
	log.Printf("Pipeline: Injection work done, returning to idle")
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
