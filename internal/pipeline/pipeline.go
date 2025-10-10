package pipeline

import (
	"context"
	"log"
	"sync"
	"sync/atomic"

	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/injection"
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
	Cancel Action = "cancel"
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
	config   *config.Config

	mu       sync.RWMutex
	wg       sync.WaitGroup
	cancel   context.CancelFunc
	stopOnce sync.Once

	running atomic.Bool
}

func New(cfg *config.Config) Pipeline {
	return &pipeline{
		actionCh: make(chan Action, 1),
		errorCh:  make(chan PipelineError, 10),
		config:   cfg,
	}
}
func (p *pipeline) Run(ctx context.Context) {
	if !p.running.CompareAndSwap(false, true) {
		log.Printf("Pipeline: Already running, ignoring Run() call")
		return
	}

	runCtx, cancel := context.WithTimeout(ctx, p.config.Recording.Timeout)
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

	recorder := recording.NewRecorder(p.config.ToRecordingConfig())
	frameCh, rErrCh, err := recorder.Start(ctx)

	if err != nil {
		log.Printf("Pipeline: Recording error: %v", err)
		p.sendError("Recording Error", "Failed to start recording", err)
		return
	}

	defer recorder.Stop()

	t, err := transcriber.NewTranscriber(p.config.ToTranscriberConfig())
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
			// Silently call an error now because on simple transcriber we just transcribe all audio when we stop, and might fail when force stop
			//p.sendError("Transcription Error", "Failed to stop transcriber cleanly", stopErr)
		}
	}()

	// Forward errors from component channels to unified pipeline error channel
	go func() {
		for err := range tErrCh {
			p.sendError("Transcription Error", "Transcription processing error", err)
		}
	}()

	go func() {
		for err := range rErrCh {
			p.sendError("Recording Error", "Recording stream error", err)
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
		return
	}

	transcriptionText, err := t.GetFinalTranscription()
	if err != nil {
		p.sendError("Transcription Error", "Failed to retrieve transcription", err)
		return
	}
	log.Printf("Pipeline: Final transcription text: %s", transcriptionText)

	injector := injection.NewInjector(p.config.ToInjectionConfig())

	if err := injector.Inject(ctx, transcriptionText); err != nil {
		p.sendError("Injection Error", "Failed to inject text", err)
	} else {
		log.Printf("Pipeline: Text injection completed successfully")
	}

	p.setStatus(Idle)
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
