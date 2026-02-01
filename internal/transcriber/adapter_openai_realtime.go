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
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// OpenAIRealtimeAdapter implements StreamingAdapter for OpenAI Realtime API transcription
type OpenAIRealtimeAdapter struct {
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

	// track current item for transcription
	currentItemID string
}

// OpenAI Realtime WebSocket message types (outgoing)
type openaiRealtimeSessionUpdate struct {
	Type    string                      `json:"type"`
	Session openaiRealtimeSessionConfig `json:"session"`
}

type openaiRealtimeSessionConfig struct {
	Modalities              []string                     `json:"modalities,omitempty"`
	InputAudioFormat        string                       `json:"input_audio_format,omitempty"`
	InputAudioTranscription *openaiRealtimeTranscription `json:"input_audio_transcription,omitempty"`
	TurnDetection           *openaiRealtimeTurnDetection `json:"turn_detection,omitempty"`
}

type openaiRealtimeTranscription struct {
	Model    string `json:"model,omitempty"`
	Language string `json:"language,omitempty"`
}

type openaiRealtimeTurnDetection struct {
	Type              string  `json:"type"`
	Threshold         float64 `json:"threshold,omitempty"`
	PrefixPaddingMs   int     `json:"prefix_padding_ms,omitempty"`
	SilenceDurationMs int     `json:"silence_duration_ms,omitempty"`
	CreateResponse    bool    `json:"create_response,omitempty"`
}

type openaiRealtimeInputAudioAppend struct {
	Type  string `json:"type"`
	Audio string `json:"audio"`
}

type openaiRealtimeInputAudioCommit struct {
	Type string `json:"type"`
}

// OpenAI Realtime WebSocket response types (incoming)
type openaiRealtimeServerEvent struct {
	Type         string                     `json:"type"`
	EventID      string                     `json:"event_id,omitempty"`
	Session      *openaiRealtimeSessionInfo `json:"session,omitempty"`
	Error        *openaiRealtimeError       `json:"error,omitempty"`
	ItemID       string                     `json:"item_id,omitempty"`
	ContentIndex int                        `json:"content_index,omitempty"`
	Transcript   string                     `json:"transcript,omitempty"`
	Delta        string                     `json:"delta,omitempty"`
}

type openaiRealtimeSessionInfo struct {
	ID    string `json:"id"`
	Model string `json:"model"`
}

type openaiRealtimeError struct {
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Param   string `json:"param,omitempty"`
}

// NewOpenAIRealtimeAdapter creates a new streaming adapter for OpenAI Realtime API
// endpoint: the WebSocket endpoint config (e.g., wss://api.openai.com, /v1/realtime)
// apiKey: OpenAI API key
// model: model ID (e.g., "gpt-4o-realtime-preview")
// lang: canonical language code (will be used for transcription config)
func NewOpenAIRealtimeAdapter(endpoint *provider.EndpointConfig, apiKey, model, lang string) *OpenAIRealtimeAdapter {
	return &OpenAIRealtimeAdapter{
		endpoint:    endpoint,
		apiKey:      apiKey,
		model:       model,
		language:    lang,
		resultsCh:   make(chan TranscriptionResult, 100),
		maxRetries:  3,
		retryDelays: defaultRetryDelays,
	}
}

// Start initiates the WebSocket connection to OpenAI Realtime API
func (a *OpenAIRealtimeAdapter) Start(ctx context.Context, lang string) error {
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

	log.Printf("openai-realtime: connected, model=%s, language=%s", a.model, a.language)
	return nil
}

// connectLocked establishes WebSocket connection and configures session. Must be called with mu held.
func (a *OpenAIRealtimeAdapter) connectLocked() error {
	wsURL, err := a.buildURL()
	if err != nil {
		return fmt.Errorf("build websocket url: %w", err)
	}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+a.apiKey)
	headers.Set("OpenAI-Beta", "realtime=v1")

	log.Printf("openai-realtime: connecting to %s", wsURL)
	conn, resp, err := websocket.DefaultDialer.DialContext(a.ctx, wsURL, headers)
	if err != nil {
		if resp != nil {
			log.Printf("openai-realtime: dial failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("websocket dial: %w", err)
	}
	a.conn = conn

	// configure session for transcription-only mode
	if err := a.configureSession(); err != nil {
		conn.Close()
		a.conn = nil
		return fmt.Errorf("configure session: %w", err)
	}

	return nil
}

// configureSession sends session.update to configure transcription mode
func (a *OpenAIRealtimeAdapter) configureSession() error {
	// configure for transcription-only mode
	// use server VAD to automatically detect speech and commit audio
	sessionUpdate := openaiRealtimeSessionUpdate{
		Type: "session.update",
		Session: openaiRealtimeSessionConfig{
			Modalities:       []string{"text"}, // text only, no audio output
			InputAudioFormat: "pcm16",          // we send 16-bit PCM
			InputAudioTranscription: &openaiRealtimeTranscription{
				Model: "gpt-4o-transcribe", // use gpt-4o for input transcription
			},
			TurnDetection: &openaiRealtimeTurnDetection{
				Type:              "server_vad",
				Threshold:         0.5,
				PrefixPaddingMs:   300,
				SilenceDurationMs: 500,
				CreateResponse:    false, // we don't want responses, just transcription
			},
		},
	}

	// add language if specified
	if a.language != "" {
		sessionUpdate.Session.InputAudioTranscription.Language = a.language
	}

	return a.conn.WriteJSON(sessionUpdate)
}

// reconnect attempts to re-establish the WebSocket connection with exponential backoff.
// Returns true if reconnection succeeded.
func (a *OpenAIRealtimeAdapter) reconnect() bool {
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
			log.Printf("openai-realtime: reconnect attempt %d/%d after %v", attempt+1, a.maxRetries, delay)

			select {
			case <-a.ctx.Done():
				return false
			case <-time.After(delay):
			}
		} else {
			log.Printf("openai-realtime: reconnect attempt %d/%d", attempt+1, a.maxRetries)
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
			log.Printf("openai-realtime: reconnected successfully")
			// notify caller of brief interruption
			select {
			case a.resultsCh <- TranscriptionResult{Error: fmt.Errorf("connection interrupted, reconnected"), IsFinal: false}:
			default:
			}
			return true
		}

		log.Printf("openai-realtime: reconnect failed: %v", err)
	}

	return false
}

// buildURL constructs the WebSocket URL with query parameters
func (a *OpenAIRealtimeAdapter) buildURL() (string, error) {
	// parse base URL and path
	baseURL := a.endpoint.BaseURL + a.endpoint.Path

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base url: %w", err)
	}

	// add model as query parameter
	q := u.Query()
	q.Set("model", a.model)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// readLoop reads messages from the WebSocket and sends results to the channel
func (a *OpenAIRealtimeAdapter) readLoop() {
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
			log.Printf("openai-realtime: read error: %v, attempting reconnection", err)
			if !a.reconnect() {
				a.resultsCh <- TranscriptionResult{Error: fmt.Errorf("websocket read: %w, reconnection failed", err)}
				return
			}
			continue
		}

		// parse message
		var event openaiRealtimeServerEvent
		if err := json.Unmarshal(message, &event); err != nil {
			log.Printf("openai-realtime: parse error: %v", err)
			continue
		}

		// handle different event types
		a.handleEvent(event)
	}
}

// handleEvent processes incoming server events
func (a *OpenAIRealtimeAdapter) handleEvent(event openaiRealtimeServerEvent) {
	switch event.Type {
	case "session.created":
		if event.Session != nil {
			log.Printf("openai-realtime: session created, id=%s, model=%s", event.Session.ID, event.Session.Model)
		}

	case "session.updated":
		log.Printf("openai-realtime: session updated")

	case "error":
		if event.Error != nil {
			errMsg := event.Error.Message
			if event.Error.Code != "" {
				errMsg = fmt.Sprintf("%s: %s", event.Error.Code, errMsg)
			}
			log.Printf("openai-realtime: error: %s", errMsg)
			a.resultsCh <- TranscriptionResult{Error: fmt.Errorf("openai: %s", errMsg)}
		}

	case "input_audio_buffer.speech_started":
		log.Printf("openai-realtime: speech started")

	case "input_audio_buffer.speech_stopped":
		log.Printf("openai-realtime: speech stopped, item_id=%s", event.ItemID)
		a.currentItemID = event.ItemID

	case "input_audio_buffer.committed":
		log.Printf("openai-realtime: audio committed, item_id=%s", event.ItemID)
		a.currentItemID = event.ItemID

	case "conversation.item.input_audio_transcription.delta":
		// partial transcription result
		if event.Delta != "" {
			a.resultsCh <- TranscriptionResult{Text: event.Delta, IsFinal: false}
		}

	case "conversation.item.input_audio_transcription.completed":
		// final transcription result
		if event.Transcript != "" {
			log.Printf("openai-realtime: transcription completed: %q", event.Transcript)
			a.resultsCh <- TranscriptionResult{Text: event.Transcript, IsFinal: true}
		}

	case "conversation.item.input_audio_transcription.failed":
		log.Printf("openai-realtime: transcription failed for item %s", event.ItemID)
		if event.Error != nil {
			a.resultsCh <- TranscriptionResult{Error: fmt.Errorf("transcription failed: %s", event.Error.Message)}
		}

	case "conversation.item.created", "conversation.item.added":
		log.Printf("openai-realtime: conversation item created/added")

	case "rate_limits.updated":
		// ignore rate limit updates

	default:
		log.Printf("openai-realtime: unhandled event type: %s", event.Type)
	}
}

// SendChunk sends audio data to the WebSocket
// OpenAI Realtime API expects base64-encoded PCM16 audio at 24kHz
// We receive 16kHz audio, so we need to resample
func (a *OpenAIRealtimeAdapter) SendChunk(audio []byte) error {
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

	// resample from 16kHz to 24kHz (OpenAI expects 24kHz)
	resampled := resample16to24(audio)

	// encode audio as base64
	audioB64 := base64.StdEncoding.EncodeToString(resampled)

	// create message
	msg := openaiRealtimeInputAudioAppend{
		Type:  "input_audio_buffer.append",
		Audio: audioB64,
	}

	// send as JSON
	a.mu.Lock()
	err := a.conn.WriteJSON(msg)
	a.mu.Unlock()

	if err != nil {
		// attempt reconnection
		log.Printf("openai-realtime: write error: %v, attempting reconnection", err)
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

// resample16to24 converts 16kHz PCM16 audio to 24kHz using linear interpolation
// Input: 16-bit PCM samples at 16kHz
// Output: 16-bit PCM samples at 24kHz
func resample16to24(input []byte) []byte {
	if len(input) < 2 {
		return input
	}

	// input has 16kHz samples (2 bytes each)
	// output needs 24kHz samples (ratio 24/16 = 1.5)
	numInputSamples := len(input) / 2
	numOutputSamples := (numInputSamples * 3) / 2

	output := make([]byte, numOutputSamples*2)

	for i := 0; i < numOutputSamples; i++ {
		// calculate position in input
		srcPos := float64(i) * 16.0 / 24.0
		srcIdx := int(srcPos)
		frac := srcPos - float64(srcIdx)

		// get source samples
		var sample1, sample2 int16
		if srcIdx*2+1 < len(input) {
			sample1 = int16(input[srcIdx*2]) | (int16(input[srcIdx*2+1]) << 8)
		}
		if (srcIdx+1)*2+1 < len(input) {
			sample2 = int16(input[(srcIdx+1)*2]) | (int16(input[(srcIdx+1)*2+1]) << 8)
		} else {
			sample2 = sample1
		}

		// linear interpolation
		outSample := int16(float64(sample1)*(1-frac) + float64(sample2)*frac)

		// write output sample (little-endian)
		output[i*2] = byte(outSample)
		output[i*2+1] = byte(outSample >> 8)
	}

	return output
}

// Results returns the channel for receiving transcription results
func (a *OpenAIRealtimeAdapter) Results() <-chan TranscriptionResult {
	return a.resultsCh
}

// Close gracefully closes the WebSocket connection
func (a *OpenAIRealtimeAdapter) Close() error {
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

	log.Printf("openai-realtime: closed")
	return nil
}
