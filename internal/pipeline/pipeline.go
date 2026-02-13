package pipeline

import (
	"context"
	"log"
	"sync"
	"sync/atomic"

	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/injection"
	"github.com/leonardotrapani/hyprvoice/internal/llm"
	"github.com/leonardotrapani/hyprvoice/internal/notify"
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
	Processing   Status = "processing" // LLM post-processing
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
	GetNotifyCh() <-chan notify.MessageType
}

// Factory types for dependency injection
type RecorderFactory func(cfg recording.Config) recording.Recorder
type TranscriberFactory func(cfg transcriber.Config) (transcriber.Transcriber, error)
type InjectorFactory func(cfg injection.Config) injection.Injector
type LLMAdapterFactory func(cfg llm.Config) (llm.Adapter, error)

// Option configures the pipeline
type Option func(*pipeline)

// WithRecorderFactory sets a custom recorder factory
func WithRecorderFactory(f RecorderFactory) Option {
	return func(p *pipeline) {
		p.recorderFactory = f
	}
}

// WithTranscriberFactory sets a custom transcriber factory
func WithTranscriberFactory(f TranscriberFactory) Option {
	return func(p *pipeline) {
		p.transcriberFactory = f
	}
}

// WithInjectorFactory sets a custom injector factory
func WithInjectorFactory(f InjectorFactory) Option {
	return func(p *pipeline) {
		p.injectorFactory = f
	}
}

// WithLLMAdapterFactory sets a custom LLM adapter factory
func WithLLMAdapterFactory(f LLMAdapterFactory) Option {
	return func(p *pipeline) {
		p.llmAdapterFactory = f
	}
}

type pipeline struct {
	status   Status
	actionCh chan Action
	errorCh  chan PipelineError
	notifyCh chan notify.MessageType
	config   *config.Config

	mu       sync.RWMutex
	wg       sync.WaitGroup
	cancel   context.CancelFunc
	stopOnce sync.Once

	running atomic.Bool

	// dependency factories (for testing)
	recorderFactory    RecorderFactory
	transcriberFactory TranscriberFactory
	injectorFactory    InjectorFactory
	llmAdapterFactory  LLMAdapterFactory
}

func New(cfg *config.Config, opts ...Option) Pipeline {
	p := &pipeline{
		actionCh: make(chan Action, 1),
		errorCh:  make(chan PipelineError, 10),
		notifyCh: make(chan notify.MessageType, 10),
		config:   cfg,
		// default factories
		recorderFactory:    recording.NewRecorder,
		transcriberFactory: transcriber.NewTranscriber,
		injectorFactory:    injection.NewInjector,
		llmAdapterFactory:  llm.NewAdapter,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
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

	recorder := p.recorderFactory(p.config.ToRecordingConfig())
	frameCh, rErrCh, err := recorder.Start(ctx)

	if err != nil {
		log.Printf("Pipeline: Recording error: %v", err)
		p.sendError("Recording Error", "Failed to start recording", err)
		return
	}

	defer recorder.Stop()

	t, err := p.transcriberFactory(p.config.ToTranscriberConfig())
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

func (p *pipeline) GetNotifyCh() <-chan notify.MessageType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.notifyCh
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

func (p *pipeline) sendNotify(mt notify.MessageType) {
	select {
	case p.notifyCh <- mt:
	default:
		log.Printf("Pipeline: Notify channel full, dropping notification")
	}
}

func (p *pipeline) handleInjectAction(ctx context.Context, recorder recording.Recorder, t transcriber.Transcriber) {
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

	// LLM post-processing phase
	textToInject := transcriptionText
	if p.config.IsLLMEnabled() {
		p.setStatus(Processing)
		p.sendNotify(notify.MsgLLMProcessing)
		log.Printf("Pipeline: LLM post-processing enabled, processing text")

		llmCfg := p.config.ToLLMConfig()
		adapter, err := p.llmAdapterFactory(llm.Config{
			Provider:          llmCfg.Provider,
			APIKey:            llmCfg.APIKey,
			Model:             llmCfg.Model,
			RemoveStutters:    llmCfg.RemoveStutters,
			AddPunctuation:    llmCfg.AddPunctuation,
			FixGrammar:        llmCfg.FixGrammar,
			RemoveFillerWords: llmCfg.RemoveFillerWords,
			CustomPrompt:      llmCfg.CustomPrompt,
			Keywords:          llmCfg.Keywords,
		})
		if err != nil {
			log.Printf("Pipeline: Failed to create LLM adapter: %v, using raw transcription", err)
		} else {
			processed, err := adapter.Process(ctx, transcriptionText)
			if err != nil {
				log.Printf("Pipeline: LLM processing failed: %v, using raw transcription", err)
			} else {
				textToInject = processed
				log.Printf("Pipeline: LLM processed text: %s", textToInject)
			}
		}
		p.setStatus(Injecting)
	}

	injector := p.injectorFactory(p.config.ToInjectionConfig())

	if err := injector.Inject(ctx, textToInject); err != nil {
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
