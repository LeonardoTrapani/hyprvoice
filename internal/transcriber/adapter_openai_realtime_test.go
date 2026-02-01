package transcriber

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// mockOpenAIRealtimeServer creates a mock WebSocket server for OpenAI Realtime API
func mockOpenAIRealtimeServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// verify auth header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("expected Bearer auth header, got: %s", auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// verify model in query
		model := r.URL.Query().Get("model")
		if model == "" {
			t.Error("expected model query parameter")
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		handler(conn)
	}))
}

func TestOpenAIRealtimeAdapter_ImplementsInterface(t *testing.T) {
	var _ StreamingAdapter = (*OpenAIRealtimeAdapter)(nil)
}

func TestOpenAIRealtimeAdapter_Start(t *testing.T) {
	var mu sync.Mutex
	sessionCreated := false
	sessionUpdated := false

	server := mockOpenAIRealtimeServer(t, func(conn *websocket.Conn) {
		// send session.created event
		sessionCreatedEvent := map[string]interface{}{
			"type":     "session.created",
			"event_id": "event_123",
			"session": map[string]interface{}{
				"id":    "sess_123",
				"model": "gpt-4o-realtime-preview",
			},
		}
		if err := conn.WriteJSON(sessionCreatedEvent); err != nil {
			t.Errorf("write session.created: %v", err)
		}
		mu.Lock()
		sessionCreated = true
		mu.Unlock()

		// read session.update from client
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var update map[string]interface{}
		if err := json.Unmarshal(msg, &update); err != nil {
			t.Errorf("unmarshal session.update: %v", err)
			return
		}

		if update["type"] != "session.update" {
			t.Errorf("expected session.update, got %s", update["type"])
		}
		mu.Lock()
		sessionUpdated = true
		mu.Unlock()

		// send session.updated response
		sessionUpdatedEvent := map[string]interface{}{
			"type":     "session.updated",
			"event_id": "event_124",
		}
		if err := conn.WriteJSON(sessionUpdatedEvent); err != nil {
			t.Errorf("write session.updated: %v", err)
		}

		// keep connection open until client closes
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})
	defer server.Close()

	// extract host for endpoint
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	endpoint := &provider.EndpointConfig{
		BaseURL: wsURL,
		Path:    "",
	}

	adapter := NewOpenAIRealtimeAdapter(endpoint, "sk-test-key", "gpt-4o-realtime-preview", "en", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := adapter.Start(ctx, "")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer adapter.Close()

	// give time for events to process
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	created := sessionCreated
	updated := sessionUpdated
	mu.Unlock()

	if !created {
		t.Error("session.created was not sent")
	}
	if !updated {
		t.Error("session.update was not received by server")
	}
}

func TestOpenAIRealtimeAdapter_SendChunk(t *testing.T) {
	var mu sync.Mutex
	receivedAudio := false

	server := mockOpenAIRealtimeServer(t, func(conn *websocket.Conn) {
		// send session.created
		sessionCreatedEvent := map[string]interface{}{
			"type": "session.created",
			"session": map[string]interface{}{
				"id": "sess_123",
			},
		}
		conn.WriteJSON(sessionCreatedEvent)

		// read session.update
		conn.ReadMessage()

		// send session.updated
		conn.WriteJSON(map[string]interface{}{"type": "session.updated"})

		// read audio chunk
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var audioMsg map[string]interface{}
		if err := json.Unmarshal(msg, &audioMsg); err != nil {
			t.Errorf("unmarshal audio: %v", err)
			return
		}

		if audioMsg["type"] == "input_audio_buffer.append" {
			audio, ok := audioMsg["audio"].(string)
			if ok && len(audio) > 0 {
				mu.Lock()
				receivedAudio = true
				mu.Unlock()
			}
		}

		// keep connection open
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	endpoint := &provider.EndpointConfig{BaseURL: wsURL, Path: ""}
	adapter := NewOpenAIRealtimeAdapter(endpoint, "sk-test", "gpt-4o-realtime-preview", "", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer adapter.Close()

	// give time for connection setup
	time.Sleep(100 * time.Millisecond)

	// send audio chunk (16kHz PCM16)
	audio := make([]byte, 320) // 10ms of 16kHz audio
	for i := range audio {
		audio[i] = byte(i % 256)
	}

	if err := adapter.SendChunk(audio); err != nil {
		t.Fatalf("SendChunk failed: %v", err)
	}

	// give time for message to be sent
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	received := receivedAudio
	mu.Unlock()

	if !received {
		t.Error("server did not receive audio chunk")
	}
}

func TestOpenAIRealtimeAdapter_TranscriptionResults(t *testing.T) {
	server := mockOpenAIRealtimeServer(t, func(conn *websocket.Conn) {
		// send session.created
		conn.WriteJSON(map[string]interface{}{"type": "session.created", "session": map[string]interface{}{"id": "sess_123"}})

		// read session.update
		conn.ReadMessage()

		// send session.updated
		conn.WriteJSON(map[string]interface{}{"type": "session.updated"})

		// simulate transcription events
		time.Sleep(50 * time.Millisecond)

		// speech started
		conn.WriteJSON(map[string]interface{}{
			"type": "input_audio_buffer.speech_started",
		})

		// partial transcription
		conn.WriteJSON(map[string]interface{}{
			"type":  "conversation.item.input_audio_transcription.delta",
			"delta": "Hello",
		})

		// more partial
		conn.WriteJSON(map[string]interface{}{
			"type":  "conversation.item.input_audio_transcription.delta",
			"delta": " world",
		})

		// final transcription
		conn.WriteJSON(map[string]interface{}{
			"type":       "conversation.item.input_audio_transcription.completed",
			"transcript": "Hello world",
		})

		// keep connection open
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	endpoint := &provider.EndpointConfig{BaseURL: wsURL, Path: ""}
	adapter := NewOpenAIRealtimeAdapter(endpoint, "sk-test", "gpt-4o-realtime-preview", "en", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer adapter.Close()

	results := adapter.Results()

	// collect results
	var partials []string
	var finals []string
	timeout := time.After(2 * time.Second)

	for {
		select {
		case result, ok := <-results:
			if !ok {
				goto done
			}
			if result.Error != nil {
				continue
			}
			if result.IsFinal {
				finals = append(finals, result.Text)
			} else {
				partials = append(partials, result.Text)
			}
			if len(finals) > 0 {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:

	if len(partials) != 2 {
		t.Errorf("expected 2 partial results, got %d: %v", len(partials), partials)
	}

	if len(finals) != 1 {
		t.Errorf("expected 1 final result, got %d: %v", len(finals), finals)
	}

	if len(finals) > 0 && finals[0] != "Hello world" {
		t.Errorf("expected final 'Hello world', got %q", finals[0])
	}
}

func TestOpenAIRealtimeAdapter_ErrorHandling(t *testing.T) {
	server := mockOpenAIRealtimeServer(t, func(conn *websocket.Conn) {
		// send session.created
		conn.WriteJSON(map[string]interface{}{"type": "session.created", "session": map[string]interface{}{"id": "sess_123"}})

		// read session.update
		conn.ReadMessage()

		// send session.updated
		conn.WriteJSON(map[string]interface{}{"type": "session.updated"})

		// send error event
		time.Sleep(50 * time.Millisecond)
		conn.WriteJSON(map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"type":    "invalid_request_error",
				"code":    "invalid_audio",
				"message": "Audio format is invalid",
			},
		})

		// keep connection open
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	endpoint := &provider.EndpointConfig{BaseURL: wsURL, Path: ""}
	adapter := NewOpenAIRealtimeAdapter(endpoint, "sk-test", "gpt-4o-realtime-preview", "", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer adapter.Close()

	results := adapter.Results()

	// wait for error
	select {
	case result := <-results:
		if result.Error == nil {
			t.Error("expected error result")
		}
		if !strings.Contains(result.Error.Error(), "invalid_audio") {
			t.Errorf("expected error containing 'invalid_audio', got: %v", result.Error)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for error result")
	}
}

func TestOpenAIRealtimeAdapter_Reconnection(t *testing.T) {
	connectCount := 0
	var mu sync.Mutex

	server := mockOpenAIRealtimeServer(t, func(conn *websocket.Conn) {
		mu.Lock()
		connectCount++
		count := connectCount
		mu.Unlock()

		// send session.created
		conn.WriteJSON(map[string]interface{}{"type": "session.created", "session": map[string]interface{}{"id": "sess_" + string(rune('0'+count))}})

		// read session.update
		conn.ReadMessage()

		// send session.updated
		conn.WriteJSON(map[string]interface{}{"type": "session.updated"})

		// first connection: close immediately to trigger reconnect
		if count == 1 {
			time.Sleep(50 * time.Millisecond)
			conn.Close()
			return
		}

		// second connection: stay open
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	endpoint := &provider.EndpointConfig{BaseURL: wsURL, Path: ""}
	adapter := NewOpenAIRealtimeAdapter(endpoint, "sk-test", "gpt-4o-realtime-preview", "", nil)
	adapter.retryDelays = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer adapter.Close()

	// wait for reconnection
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	finalCount := connectCount
	mu.Unlock()

	if finalCount < 2 {
		t.Errorf("expected at least 2 connections (reconnection), got %d", finalCount)
	}
}

func TestOpenAIRealtimeAdapter_Close(t *testing.T) {
	server := mockOpenAIRealtimeServer(t, func(conn *websocket.Conn) {
		conn.WriteJSON(map[string]interface{}{"type": "session.created", "session": map[string]interface{}{"id": "sess_123"}})
		conn.ReadMessage()
		conn.WriteJSON(map[string]interface{}{"type": "session.updated"})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	endpoint := &provider.EndpointConfig{BaseURL: wsURL, Path: ""}
	adapter := NewOpenAIRealtimeAdapter(endpoint, "sk-test", "gpt-4o-realtime-preview", "", nil)

	ctx := context.Background()

	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// close should not block or panic
	err := adapter.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	// results channel should be closed
	select {
	case _, ok := <-adapter.Results():
		if ok {
			// drain any remaining results
			for range adapter.Results() {
			}
		}
	case <-time.After(time.Second):
		t.Error("results channel not closed after Close()")
	}
}

func TestResample16to24(t *testing.T) {
	// test with simple audio data
	input := make([]byte, 32) // 16 samples at 16kHz
	for i := 0; i < 16; i++ {
		// write sample value (little-endian)
		sample := int16(i * 1000)
		input[i*2] = byte(sample)
		input[i*2+1] = byte(sample >> 8)
	}

	output := resample16to24(input)

	// 16 samples at 16kHz = 24 samples at 24kHz (ratio 1.5)
	expectedSamples := 24
	if len(output) != expectedSamples*2 {
		t.Errorf("expected %d bytes, got %d", expectedSamples*2, len(output))
	}

	// output should have reasonable values (interpolated)
	for i := 0; i < expectedSamples; i++ {
		sample := int16(output[i*2]) | (int16(output[i*2+1]) << 8)
		if sample < -32768 || sample > 32767 {
			t.Errorf("sample %d out of range: %d", i, sample)
		}
	}
}

func TestResample16to24_EmptyInput(t *testing.T) {
	output := resample16to24([]byte{})
	if len(output) != 0 {
		t.Errorf("expected empty output for empty input, got %d bytes", len(output))
	}
}

func TestResample16to24_SingleSample(t *testing.T) {
	input := []byte{0x00, 0x10} // single sample
	output := resample16to24(input)
	// with only 1 sample, output should be minimal
	if len(output) == 0 {
		t.Error("expected non-empty output for single sample")
	}
}
