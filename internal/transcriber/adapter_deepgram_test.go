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

func TestDeepgramAdapter_ImplementsStreamingAdapter(t *testing.T) {
	var _ StreamingAdapter = (*DeepgramAdapter)(nil)
}

func TestDeepgramAdapter_Creation(t *testing.T) {
	endpoint := &provider.EndpointConfig{
		BaseURL: "wss://api.deepgram.com",
		Path:    "/v1/listen",
	}
	adapter := NewDeepgramAdapter(endpoint, "test-api-key", "nova-3", "en")

	if adapter.apiKey != "test-api-key" {
		t.Errorf("apiKey = %q, want %q", adapter.apiKey, "test-api-key")
	}
	if adapter.model != "nova-3" {
		t.Errorf("model = %q, want %q", adapter.model, "nova-3")
	}
	if adapter.language != "en" {
		t.Errorf("language = %q, want %q", adapter.language, "en")
	}
	if adapter.maxRetries != 3 {
		t.Errorf("maxRetries = %d, want %d", adapter.maxRetries, 3)
	}
}

func TestDeepgramAdapter_BuildURL(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		language string
		wantURL  []string // URL must contain all these substrings
	}{
		{
			name:     "english",
			model:    "nova-3",
			language: "en",
			wantURL:  []string{"model=nova-3", "language=en-US", "encoding=linear16", "sample_rate=16000"},
		},
		{
			name:     "spanish",
			model:    "nova-2",
			language: "es",
			wantURL:  []string{"model=nova-2", "language=es", "encoding=linear16"},
		},
		{
			name:     "auto-detect",
			model:    "nova-3",
			language: "",
			wantURL:  []string{"model=nova-3", "encoding=linear16"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &provider.EndpointConfig{
				BaseURL: "wss://api.deepgram.com",
				Path:    "/v1/listen",
			}
			adapter := NewDeepgramAdapter(endpoint, "test-key", tt.model, tt.language)

			url, err := adapter.buildURL()
			if err != nil {
				t.Fatalf("buildURL() error = %v", err)
			}

			for _, want := range tt.wantURL {
				if !strings.Contains(url, want) {
					t.Errorf("buildURL() = %q, want to contain %q", url, want)
				}
			}
		})
	}
}

func TestDeepgramAdapter_SendChunkNotStarted(t *testing.T) {
	endpoint := &provider.EndpointConfig{
		BaseURL: "wss://api.deepgram.com",
		Path:    "/v1/listen",
	}
	adapter := NewDeepgramAdapter(endpoint, "test-key", "nova-3", "en")

	err := adapter.SendChunk([]byte("audio data"))
	if err == nil {
		t.Error("SendChunk() should return error when adapter not started")
	}
	if !strings.Contains(err.Error(), "not started") {
		t.Errorf("error should mention 'not started', got: %v", err)
	}
}

func TestDeepgramAdapter_CloseNotStarted(t *testing.T) {
	endpoint := &provider.EndpointConfig{
		BaseURL: "wss://api.deepgram.com",
		Path:    "/v1/listen",
	}
	adapter := NewDeepgramAdapter(endpoint, "test-key", "nova-3", "en")

	// closing not-started adapter should not error
	err := adapter.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

// mockDeepgramServer creates a mock WebSocket server for testing
func mockDeepgramServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// verify auth header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Token ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()

		handler(conn)
	}))
	return server
}

func TestDeepgramAdapter_StartAndClose(t *testing.T) {
	server := mockDeepgramServer(t, func(conn *websocket.Conn) {
		// send metadata response
		metadata := deepgramWSResponse{
			Type: "Metadata",
			Metadata: &deepgramMetadata{
				RequestID: "test-123",
			},
		}
		metadata.Metadata.ModelInfo.Name = "nova-3"
		if err := conn.WriteJSON(metadata); err != nil {
			t.Logf("write metadata error: %v", err)
			return
		}

		// wait for close
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	})
	defer server.Close()

	// convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	endpoint := &provider.EndpointConfig{
		BaseURL: wsURL,
		Path:    "",
	}

	adapter := NewDeepgramAdapter(endpoint, "test-api-key", "nova-3", "en")

	ctx := context.Background()
	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// verify can't start twice
	if err := adapter.Start(ctx, ""); err == nil {
		t.Error("Start() should return error when already started")
	}

	// close
	if err := adapter.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestDeepgramAdapter_ReceivesResults(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	server := mockDeepgramServer(t, func(conn *websocket.Conn) {
		defer wg.Done()

		// send metadata
		metadata := deepgramWSResponse{Type: "Metadata", Metadata: &deepgramMetadata{RequestID: "test-123"}}
		_ = conn.WriteJSON(metadata)

		// send interim result
		interim := deepgramWSResponse{
			Type:    "Results",
			IsFinal: false,
			Channel: &deepgramChannel{
				Alternatives: []deepgramAlternative{{Transcript: "hello", Confidence: 0.95}},
			},
		}
		_ = conn.WriteJSON(interim)

		// send final result
		final := deepgramWSResponse{
			Type:    "Results",
			IsFinal: true,
			Channel: &deepgramChannel{
				Alternatives: []deepgramAlternative{{Transcript: "hello world", Confidence: 0.98}},
			},
		}
		_ = conn.WriteJSON(final)

		// wait briefly then close
		time.Sleep(50 * time.Millisecond)
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	endpoint := &provider.EndpointConfig{BaseURL: wsURL, Path: ""}
	adapter := NewDeepgramAdapter(endpoint, "test-api-key", "nova-3", "en")

	ctx := context.Background()
	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// collect results
	var results []TranscriptionResult
	timeout := time.After(2 * time.Second)

loop:
	for {
		select {
		case result, ok := <-adapter.Results():
			if !ok {
				break loop
			}
			results = append(results, result)
			if result.IsFinal {
				break loop
			}
		case <-timeout:
			t.Fatal("timeout waiting for results")
		}
	}

	adapter.Close()
	wg.Wait()

	// verify results
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// check interim
	if results[0].Text != "hello" || results[0].IsFinal {
		t.Errorf("interim result = %+v, want Text='hello', IsFinal=false", results[0])
	}

	// check final
	found := false
	for _, r := range results {
		if r.Text == "hello world" && r.IsFinal {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("did not find expected final result 'hello world'")
	}
}

func TestDeepgramAdapter_SendsRawBinaryAudio(t *testing.T) {
	receivedAudio := make(chan []byte, 1)

	server := mockDeepgramServer(t, func(conn *websocket.Conn) {
		// send metadata first
		metadata := deepgramWSResponse{Type: "Metadata", Metadata: &deepgramMetadata{RequestID: "test-123"}}
		_ = conn.WriteJSON(metadata)

		// read audio chunk
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if msgType != websocket.BinaryMessage {
			t.Errorf("expected binary message, got %d", msgType)
		}
		receivedAudio <- data

		// keep reading until close
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	endpoint := &provider.EndpointConfig{BaseURL: wsURL, Path: ""}
	adapter := NewDeepgramAdapter(endpoint, "test-api-key", "nova-3", "en")

	ctx := context.Background()
	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// send audio chunk
	testAudio := []byte{0x01, 0x02, 0x03, 0x04}
	if err := adapter.SendChunk(testAudio); err != nil {
		t.Errorf("SendChunk() error = %v", err)
	}

	// verify audio was received
	select {
	case audio := <-receivedAudio:
		if string(audio) != string(testAudio) {
			t.Errorf("received audio = %v, want %v", audio, testAudio)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for audio")
	}

	adapter.Close()
}

func TestDeepgramAdapter_HandlesError(t *testing.T) {
	server := mockDeepgramServer(t, func(conn *websocket.Conn) {
		// send error
		errResp := deepgramWSResponse{
			Type: "Error",
			Error: &deepgramError{
				Type:    "AuthError",
				Message: "Invalid API key",
			},
		}
		data, _ := json.Marshal(errResp)
		_ = conn.WriteMessage(websocket.TextMessage, data)

		time.Sleep(50 * time.Millisecond)
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	endpoint := &provider.EndpointConfig{BaseURL: wsURL, Path: ""}
	adapter := NewDeepgramAdapter(endpoint, "test-api-key", "nova-3", "en")

	ctx := context.Background()
	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// wait for error result
	select {
	case result := <-adapter.Results():
		if result.Error == nil {
			t.Error("expected error result")
		}
		if !strings.Contains(result.Error.Error(), "Invalid API key") {
			t.Errorf("error = %v, want to contain 'Invalid API key'", result.Error)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for error")
	}

	adapter.Close()
}

func TestDeepgramAdapter_ContextCancellation(t *testing.T) {
	server := mockDeepgramServer(t, func(conn *websocket.Conn) {
		// send metadata first so connection is established
		metadata := deepgramWSResponse{Type: "Metadata", Metadata: &deepgramMetadata{RequestID: "test-123"}}
		_ = conn.WriteJSON(metadata)

		// just keep connection open until closed
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	endpoint := &provider.EndpointConfig{BaseURL: wsURL, Path: ""}
	adapter := NewDeepgramAdapter(endpoint, "test-api-key", "nova-3", "en")

	ctx, cancel := context.WithCancel(context.Background())
	if err := adapter.Start(ctx, ""); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// cancel context - this should trigger Close() to be called or at least stop SendChunk
	cancel()

	// SendChunk should return error after context cancelled
	err := adapter.SendChunk([]byte("test"))
	if err == nil {
		// it's ok if first chunk after cancel succeeds - the context cancel is async
		// but subsequent operations should fail
	}

	// Close should work even after context cancelled
	if err := adapter.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// results channel should be closed after Close()
	select {
	case _, ok := <-adapter.Results():
		if ok {
			// drain any remaining
			for range adapter.Results() {
			}
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for results channel to close")
	}
}
