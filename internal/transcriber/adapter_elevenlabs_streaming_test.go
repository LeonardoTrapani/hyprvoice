package transcriber

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// mockElevenLabsServer creates a test WebSocket server that simulates ElevenLabs
func mockElevenLabsServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check API key header
		apiKey := r.Header.Get("xi-api-key")
		if apiKey == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}

		// upgrade to websocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()

		handler(conn)
	}))
}

func TestElevenLabsStreamingAdapter_ImplementsInterface(t *testing.T) {
	var _ StreamingAdapter = (*ElevenLabsStreamingAdapter)(nil)
}

func TestElevenLabsStreamingAdapter_Start(t *testing.T) {
	server := mockElevenLabsServer(t, func(conn *websocket.Conn) {
		// send session started
		msg := elevenLabsWSMessage{
			MessageType: "session_started",
			SessionID:   "test-session-123",
		}
		conn.WriteJSON(msg)

		// keep connection open
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})
	defer server.Close()

	// convert http://... to ws://...
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	adapter := NewElevenLabsStreamingAdapter(
		&provider.EndpointConfig{BaseURL: wsURL, Path: ""},
		"test-api-key",
		"scribe_v1",
		"en",
	)

	ctx := context.Background()
	err := adapter.Start(ctx, "")
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// give time for session_started to be received
	time.Sleep(50 * time.Millisecond)

	err = adapter.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestElevenLabsStreamingAdapter_SendChunk(t *testing.T) {
	receivedChunks := make(chan []byte, 10)

	server := mockElevenLabsServer(t, func(conn *websocket.Conn) {
		// send session started
		conn.WriteJSON(elevenLabsWSMessage{
			MessageType: "session_started",
			SessionID:   "test-session",
		})

		// read incoming messages
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var msg elevenLabsInputAudioChunk
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			if msg.MessageType == "input_audio_chunk" {
				decoded, _ := base64.StdEncoding.DecodeString(msg.AudioBase64)
				receivedChunks <- decoded
			}
		}
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	adapter := NewElevenLabsStreamingAdapter(
		&provider.EndpointConfig{BaseURL: wsURL, Path: ""},
		"test-api-key",
		"scribe_v1",
		"en",
	)

	ctx := context.Background()
	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer adapter.Close()

	// send audio chunk
	testAudio := []byte{0x01, 0x02, 0x03, 0x04}
	if err := adapter.SendChunk(testAudio); err != nil {
		t.Fatalf("SendChunk() error: %v", err)
	}

	// verify received
	select {
	case received := <-receivedChunks:
		if string(received) != string(testAudio) {
			t.Errorf("received audio mismatch: got %v, want %v", received, testAudio)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for audio chunk")
	}
}

func TestElevenLabsStreamingAdapter_Results(t *testing.T) {
	server := mockElevenLabsServer(t, func(conn *websocket.Conn) {
		// send session started
		conn.WriteJSON(elevenLabsWSMessage{
			MessageType: "session_started",
			SessionID:   "test-session",
		})

		// send partial transcript
		conn.WriteJSON(elevenLabsWSMessage{
			MessageType: "partial_transcript",
			Text:        "hello",
		})

		// send committed transcript
		conn.WriteJSON(elevenLabsWSMessage{
			MessageType: "committed_transcript",
			Text:        "hello world",
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

	adapter := NewElevenLabsStreamingAdapter(
		&provider.EndpointConfig{BaseURL: wsURL, Path: ""},
		"test-api-key",
		"scribe_v1",
		"en",
	)

	ctx := context.Background()
	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer adapter.Close()

	results := adapter.Results()

	// check partial result
	select {
	case result := <-results:
		if result.Error != nil {
			t.Fatalf("unexpected error: %v", result.Error)
		}
		if result.Text != "hello" {
			t.Errorf("partial text: got %q, want %q", result.Text, "hello")
		}
		if result.IsFinal {
			t.Error("partial result should not be final")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for partial result")
	}

	// check final result
	select {
	case result := <-results:
		if result.Error != nil {
			t.Fatalf("unexpected error: %v", result.Error)
		}
		if result.Text != "hello world" {
			t.Errorf("final text: got %q, want %q", result.Text, "hello world")
		}
		if !result.IsFinal {
			t.Error("committed result should be final")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for final result")
	}
}

func TestElevenLabsStreamingAdapter_ErrorMessages(t *testing.T) {
	server := mockElevenLabsServer(t, func(conn *websocket.Conn) {
		// send session started
		conn.WriteJSON(elevenLabsWSMessage{
			MessageType: "session_started",
			SessionID:   "test-session",
		})

		// send error
		conn.WriteJSON(elevenLabsWSMessage{
			MessageType: "error",
			Error:       "test error message",
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

	adapter := NewElevenLabsStreamingAdapter(
		&provider.EndpointConfig{BaseURL: wsURL, Path: ""},
		"test-api-key",
		"scribe_v1",
		"en",
	)

	ctx := context.Background()
	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer adapter.Close()

	results := adapter.Results()

	// check error result
	select {
	case result := <-results:
		if result.Error == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(result.Error.Error(), "test error message") {
			t.Errorf("error message: got %q, want to contain %q", result.Error.Error(), "test error message")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for error result")
	}
}

func TestElevenLabsStreamingAdapter_LanguageConversion(t *testing.T) {
	var receivedURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()

		// check API key header
		if r.Header.Get("xi-api-key") == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		conn.WriteJSON(elevenLabsWSMessage{
			MessageType: "session_started",
			SessionID:   "test-session",
		})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	adapter := NewElevenLabsStreamingAdapter(
		&provider.EndpointConfig{BaseURL: wsURL, Path: "/v1/speech-to-text/realtime"},
		"test-api-key",
		"scribe_v1",
		"es", // Spanish
	)

	ctx := context.Background()
	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	adapter.Close()

	// verify language_code was set
	if !strings.Contains(receivedURL, "language_code=es") {
		t.Errorf("URL should contain language_code=es, got: %s", receivedURL)
	}

	// verify model_id was set
	if !strings.Contains(receivedURL, "model_id=scribe_v1") {
		t.Errorf("URL should contain model_id=scribe_v1, got: %s", receivedURL)
	}

	// verify audio_format was set
	if !strings.Contains(receivedURL, "audio_format=pcm_16000") {
		t.Errorf("URL should contain audio_format=pcm_16000, got: %s", receivedURL)
	}
}

func TestElevenLabsStreamingAdapter_Close(t *testing.T) {
	server := mockElevenLabsServer(t, func(conn *websocket.Conn) {
		conn.WriteJSON(elevenLabsWSMessage{
			MessageType: "session_started",
			SessionID:   "test-session",
		})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	adapter := NewElevenLabsStreamingAdapter(
		&provider.EndpointConfig{BaseURL: wsURL, Path: ""},
		"test-api-key",
		"scribe_v1",
		"en",
	)

	ctx := context.Background()
	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// close should not block
	done := make(chan struct{})
	go func() {
		adapter.Close()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("Close() blocked for too long")
	}

	// results channel should be closed
	_, ok := <-adapter.Results()
	if ok {
		// there might be buffered results, drain them
		for range adapter.Results() {
		}
	}
}

func TestElevenLabsStreamingAdapter_NotStarted(t *testing.T) {
	adapter := NewElevenLabsStreamingAdapter(
		&provider.EndpointConfig{BaseURL: "wss://api.elevenlabs.io", Path: "/v1/speech-to-text/realtime"},
		"test-api-key",
		"scribe_v1",
		"en",
	)

	// SendChunk should fail when not started
	err := adapter.SendChunk([]byte{0x01, 0x02})
	if err == nil {
		t.Error("SendChunk() should fail when adapter not started")
	}
	if !strings.Contains(err.Error(), "not started") {
		t.Errorf("error should mention 'not started', got: %v", err)
	}

	// Close should not fail when not started
	err = adapter.Close()
	if err != nil {
		t.Errorf("Close() should not fail when not started: %v", err)
	}
}
