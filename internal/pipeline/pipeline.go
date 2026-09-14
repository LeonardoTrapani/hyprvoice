package pipeline

import (
	"context"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

// StatusEvent is a point-in-time snapshot of the pipeline, published to
// subscribers of the daemon's status stream.
type StatusEvent struct {
	Status Status `json:"status"`

	// Listening reports whether the microphone is currently open. It is
	// deliberately separate from Status: with a streaming transcriber the
	// pipeline sits in Transcribing for the whole time the user is speaking,
	// so Status alone cannot tell an indicator when to show "listening".
	Listening bool `json:"listening"`

	// Levels carries normalised 0..1 loudness values covering the audio since
	// the previous event, oldest first. Empty on a status change.
	Levels []float64 `json:"levels,omitempty"`

	At time.Time `json:"at"`
}

type Pipeline interface {
	Run(ctx context.Context)
	Stop()
	Status() Status

	// Listening reports whether the microphone is open, which Status cannot
	// express on its own -- see StatusEvent.Listening.
	Listening() bool

	GetActionCh() chan<- Action
	GetErrorCh() <-chan PipelineError
	GetNotifyCh() <-chan notify.MessageType
	GetEventCh() <-chan StatusEvent

	// SetLevelsWanted controls whether audio loudness is measured while
	// recording. The daemon enables it only while something is subscribed to
	// the status stream, so an unwatched pipeline does no extra work.
	SetLevelsWanted(bool)
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
	eventCh  chan StatusEvent
	config   *config.Config

	mu       sync.RWMutex
	wg       sync.WaitGroup
	cancel   context.CancelFunc
	stopOnce sync.Once

	running      atomic.Bool
	listening    atomic.Bool
	levelsWanted atomic.Bool

	// dependency factories (for testing)
	recorderFactory    RecorderFactory
	transcriberFactory TranscriberFactory
	injectorFactory    InjectorFactory
	llmAdapterFactory  LLMAdapterFactory
}

func New(cfg *config.Config, opts ...Option) Pipeline {
	p := &pipeline{
		// Explicit, so a pipeline that has been constructed but not yet run
		// reports idle rather than the empty string.
		status:   Idle,
		actionCh: make(chan Action, 1),
		errorCh:  make(chan PipelineError, 10),
		notifyCh: make(chan notify.MessageType, 10),
		// Deep enough to absorb a burst of level events while a slow
		// subscriber is being served; sends drop rather than block, so a
		// stalled reader can never hold up the audio path.
		eventCh: make(chan StatusEvent, 64),
		config:  cfg,
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
		p.setListening(false)
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
	p.setListening(true)

	// Measure loudness on the way past. The tap forwards every frame
	// untouched; it exists so an indicator can show a real waveform instead of
	// an animation pretending to be the user's voice.
	recCfg := p.config.ToRecordingConfig()
	frameCh = tapLevels(
		ctx, frameCh,
		recCfg.ChannelBufferSize, recCfg.SampleRate, recCfg.Channels,
		p.levelsWanted.Load,
		func(levels []float64) {
			p.emit(StatusEvent{
				Status:    p.Status(),
				Listening: true,
				Levels:    levels,
				At:        time.Now(),
			})
		},
	)

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
	p.status = status
	p.mu.Unlock()

	p.emit(StatusEvent{Status: status, Listening: p.listening.Load(), At: time.Now()})
}

// emit publishes an event, discarding it if no subscriber is keeping up.
// Status is always readable via the Status() snapshot, so a dropped event
// costs a frame of animation, never correctness.
func (p *pipeline) emit(ev StatusEvent) {
	select {
	case p.eventCh <- ev:
	default:
	}
}

func (p *pipeline) GetEventCh() <-chan StatusEvent {
	return p.eventCh
}

func (p *pipeline) Listening() bool {
	return p.listening.Load()
}

func (p *pipeline) SetLevelsWanted(wanted bool) {
	p.levelsWanted.Store(wanted)
}

// setListening records whether the microphone is open and republishes the
// current status so subscribers see the transition.
func (p *pipeline) setListening(listening bool) {
	if p.listening.Swap(listening) == listening {
		return
	}
	p.emit(StatusEvent{Status: p.Status(), Listening: listening, At: time.Now()})
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
	p.setListening(false)

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

	// Sanitize: replace line-terminating characters with spaces to prevent
	// unintended Enter keypresses during injection, which can submit forms mid-sentence.
	// Covers ASCII controls (\r, \n, \v, \f), Unicode NEL (U+0085),
	// LINE SEPARATOR (U+2028), and PARAGRAPH SEPARATOR (U+2029).
	textToInject = strings.Map(func(r rune) rune {
		switch r {
		case '\r', '\n', '\v', '\f', '\u0085', '\u2028', '\u2029':
			return ' '
		}
		return r
	}, textToInject)

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
