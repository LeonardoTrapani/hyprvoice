package transcriber

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leonardotrapani/hyprvoice/internal/language"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// default retry delays for reconnection (exponential backoff: 1s, 2s, 4s)
var defaultRetryDelays = []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

// ElevenLabsStreamingAdapter implements StreamingAdapter for ElevenLabs real-time transcription
type ElevenLabsStreamingAdapter struct {
	endpoint  *provider.EndpointConfig
	apiKey    string
	model     string
	language  string
	conn      *websocket.Conn
	resultsCh chan TranscriptionResult
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	started   bool

	// reconnection config
	maxRetries  int
	retryDelays []time.Duration

	// finalization signaling
	commitDone chan struct{}
}

// ElevenLabs WebSocket message types (outgoing)
type elevenLabsInputAudioChunk struct {
	MessageType string `json:"message_type"`
	AudioBase64 string `json:"audio_base_64"`
	Commit      bool   `json:"commit"`
	SampleRate  int    `json:"sample_rate"`
}

// ElevenLabs WebSocket response types (incoming)
type elevenLabsWSMessage struct {
	MessageType  string `json:"message_type"`
	Text         string `json:"text,omitempty"`
	Error        string `json:"error,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

// NewElevenLabsStreamingAdapter creates a new streaming adapter for ElevenLabs
// endpoint: the WebSocket endpoint config (e.g., wss://api.elevenlabs.io, /v1/speech-to-text/realtime)
// apiKey: ElevenLabs API key
// model: model ID (e.g., "scribe_v1")
// lang: canonical language code (will be converted to provider format)
func NewElevenLabsStreamingAdapter(endpoint *provider.EndpointConfig, apiKey, model, lang string) *ElevenLabsStreamingAdapter {
	return &ElevenLabsStreamingAdapter{
		endpoint:    endpoint,
		apiKey:      apiKey,
		model:       model,
		language:    lang,
		resultsCh:   make(chan TranscriptionResult, 100),
		maxRetries:  3,
		retryDelays: defaultRetryDelays,
		commitDone:  make(chan struct{}, 1),
	}
}

// Start initiates the WebSocket connection to ElevenLabs
func (a *ElevenLabsStreamingAdapter) Start(ctx context.Context, lang string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.started {
		return fmt.Errorf("adapter already started")
	}

	// use lang param if provided, otherwise use constructor lang
	if lang != "" {
		a.language = lang
	}

	// create cancelable context
	a.ctx, a.cancel = context.WithCancel(ctx)

	// connect to WebSocket
	if err := a.connectLocked(); err != nil {
		return err
	}
	a.started = true

	// start reader goroutine
	a.wg.Add(1)
	go a.readLoop()

	log.Printf("elevenlabs-streaming: connected, model=%s, language=%s", a.model, a.language)
	return nil
}

// connectLocked establishes WebSocket connection. Must be called with mu held.
func (a *ElevenLabsStreamingAdapter) connectLocked() error {
	wsURL, err := a.buildURL()
	if err != nil {
		return fmt.Errorf("build websocket url: %w", err)
	}

	headers := http.Header{}
	headers.Set("xi-api-key", a.apiKey)

	log.Printf("elevenlabs-streaming: connecting to %s", wsURL)
	conn, resp, err := websocket.DefaultDialer.DialContext(a.ctx, wsURL, headers)
	if err != nil {
		if resp != nil {
			log.Printf("elevenlabs-streaming: dial failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("websocket dial: %w", err)
	}
	a.conn = conn
	return nil
}

// reconnect attempts to re-establish the WebSocket connection with exponential backoff.
// Returns true if reconnection succeeded.
func (a *ElevenLabsStreamingAdapter) reconnect() bool {
	for attempt := 0; attempt < a.maxRetries; attempt++ {
		// check if context cancelled
		select {
		case <-a.ctx.Done():
			return false
		default:
		}

		// wait before retry (skip wait on first attempt)
		if attempt > 0 {
			delay := a.retryDelays[attempt-1]
			if attempt-1 >= len(a.retryDelays) {
				delay = a.retryDelays[len(a.retryDelays)-1]
			}
			log.Printf("elevenlabs-streaming: reconnect attempt %d/%d after %v", attempt+1, a.maxRetries, delay)

			select {
			case <-a.ctx.Done():
				return false
			case <-time.After(delay):
			}
		} else {
			log.Printf("elevenlabs-streaming: reconnect attempt %d/%d", attempt+1, a.maxRetries)
		}

		a.mu.Lock()
		// close old connection if exists
		if a.conn != nil {
			a.conn.Close()
			a.conn = nil
		}

		err := a.connectLocked()
		a.mu.Unlock()

		if err == nil {
			log.Printf("elevenlabs-streaming: reconnected successfully")
			// notify caller of brief interruption
			select {
			case a.resultsCh <- TranscriptionResult{Error: fmt.Errorf("connection interrupted, reconnected"), IsFinal: false}:
			default:
			}
			return true
		}

		log.Printf("elevenlabs-streaming: reconnect failed: %v", err)
	}

	return false
}

// buildURL constructs the WebSocket URL with query parameters
func (a *ElevenLabsStreamingAdapter) buildURL() (string, error) {
	// parse base URL and path
	baseURL := a.endpoint.BaseURL + a.endpoint.Path

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base url: %w", err)
	}

	// add query parameters
	q := u.Query()
	q.Set("model_id", a.model)
	q.Set("audio_format", "pcm_16000") // we use 16kHz PCM

	// add language if specified
	providerLang := language.ToProviderFormat(a.language, "elevenlabs")
	if providerLang != "" {
		q.Set("language_code", providerLang)
	}

	// use VAD for automatic commit (easier for real-time use)
	q.Set("commit_strategy", "vad")

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// readLoop reads messages from the WebSocket and sends results to the channel
func (a *ElevenLabsStreamingAdapter) readLoop() {
	defer a.wg.Done()
	defer close(a.resultsCh)

	for {
		select {
		case <-a.ctx.Done():
			return
		default:
		}

		a.mu.Lock()
		conn := a.conn
		a.mu.Unlock()

		if conn == nil {
			// no connection, try to reconnect
			if !a.reconnect() {
				a.resultsCh <- TranscriptionResult{Error: fmt.Errorf("connection lost, reconnection failed after %d attempts", a.maxRetries)}
				return
			}
			continue
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			// check if context was cancelled (normal shutdown)
			select {
			case <-a.ctx.Done():
				return
			default:
			}

			// attempt reconnection
			log.Printf("elevenlabs-streaming: read error: %v, attempting reconnection", err)
			if !a.reconnect() {
				a.resultsCh <- TranscriptionResult{Error: fmt.Errorf("websocket read: %w, reconnection failed", err)}
				return
			}
			continue
		}

		// parse message
		var msg elevenLabsWSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("elevenlabs-streaming: parse error: %v", err)
			continue
		}

		// handle different message types
		switch msg.MessageType {
		case "session_started":
			log.Printf("elevenlabs-streaming: session started, id=%s", msg.SessionID)

		case "partial_transcript":
			// interim result
			if msg.Text != "" {
				a.resultsCh <- TranscriptionResult{Text: msg.Text, IsFinal: false}
			}

		case "committed_transcript", "committed_transcript_with_timestamps":
			// final result
			log.Printf("elevenlabs-streaming: committed: %q", msg.Text)
			if msg.Text != "" {
				a.resultsCh <- TranscriptionResult{Text: msg.Text, IsFinal: true}
			}
			// signal finalization is done (non-blocking)
			select {
			case a.commitDone <- struct{}{}:
			default:
			}

		case "error", "auth_error", "quota_exceeded", "rate_limited",
			"queue_overflow", "resource_exhausted", "session_time_limit_exceeded",
			"input_error", "chunk_size_exceeded", "insufficient_audio_activity",
			"transcriber_error", "commit_throttled", "unaccepted_terms":
			// error message
			errMsg := msg.Error
			if errMsg == "" {
				errMsg = msg.MessageType
			}
			log.Printf("elevenlabs-streaming: error: %s", errMsg)
			a.resultsCh <- TranscriptionResult{Error: fmt.Errorf("elevenlabs: %s", errMsg)}

		default:
			log.Printf("elevenlabs-streaming: unknown message type: %s", msg.MessageType)
		}
	}
}

// SendChunk sends audio data to the WebSocket
func (a *ElevenLabsStreamingAdapter) SendChunk(audio []byte) error {
	a.mu.Lock()
	if !a.started {
		a.mu.Unlock()
		return fmt.Errorf("adapter not started")
	}
	conn := a.conn
	a.mu.Unlock()

	// check context
	select {
	case <-a.ctx.Done():
		return a.ctx.Err()
	default:
	}

	if conn == nil {
		return fmt.Errorf("no connection")
	}

	// encode audio as base64
	audioB64 := base64.StdEncoding.EncodeToString(audio)

	// create message
	msg := elevenLabsInputAudioChunk{
		MessageType: "input_audio_chunk",
		AudioBase64: audioB64,
		Commit:      false, // let VAD handle commits
		SampleRate:  16000,
	}

	// send as JSON
	a.mu.Lock()
	err := a.conn.WriteJSON(msg)
	a.mu.Unlock()

	if err != nil {
		// attempt reconnection
		log.Printf("elevenlabs-streaming: write error: %v, attempting reconnection", err)
		if a.reconnect() {
			// retry the chunk after reconnection
			a.mu.Lock()
			err = a.conn.WriteJSON(msg)
			a.mu.Unlock()
			if err == nil {
				return nil
			}
		}
		return fmt.Errorf("websocket write: %w", err)
	}

	return nil
}

// Results returns the channel for receiving transcription results
func (a *ElevenLabsStreamingAdapter) Results() <-chan TranscriptionResult {
	return a.resultsCh
}

// Finalize sends a commit message to force ElevenLabs to commit any pending audio
// and waits for the committed_transcript response
func (a *ElevenLabsStreamingAdapter) Finalize(ctx context.Context) error {
	a.mu.Lock()
	if !a.started {
		a.mu.Unlock()
		return nil
	}
	conn := a.conn
	a.mu.Unlock()

	if conn == nil {
		return nil
	}

	// drain any previous commit signals
	select {
	case <-a.commitDone:
	default:
	}

	// send empty audio chunk with commit=true to force finalization
	msg := elevenLabsInputAudioChunk{
		MessageType: "input_audio_chunk",
		AudioBase64: "",
		Commit:      true,
		SampleRate:  16000,
	}

	a.mu.Lock()
	err := a.conn.WriteJSON(msg)
	a.mu.Unlock()

	if err != nil {
		log.Printf("elevenlabs-streaming: finalize write error: %v", err)
		return fmt.Errorf("finalize write: %w", err)
	}

	log.Printf("elevenlabs-streaming: sent commit, waiting for final transcript")

	// wait for committed_transcript or timeout
	select {
	case <-a.commitDone:
		log.Printf("elevenlabs-streaming: finalize complete")
		return nil
	case <-ctx.Done():
		log.Printf("elevenlabs-streaming: finalize timeout")
		return ctx.Err()
	case <-a.ctx.Done():
		return a.ctx.Err()
	}
}

// Close gracefully closes the WebSocket connection
func (a *ElevenLabsStreamingAdapter) Close() error {
	a.mu.Lock()

	if !a.started {
		a.mu.Unlock()
		return nil
	}

	// cancel context first to signal reader to stop
	if a.cancel != nil {
		a.cancel()
	}

	// get conn ref while holding lock
	conn := a.conn

	a.started = false
	a.mu.Unlock()

	// close websocket outside of lock (readLoop may be blocked on read)
	if conn != nil {
		// send close frame (best effort)
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		conn.Close()
	}

	// wait for reader to finish
	a.wg.Wait()

	log.Printf("elevenlabs-streaming: closed")
	return nil
}
