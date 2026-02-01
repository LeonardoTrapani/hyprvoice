package transcriber

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
	"github.com/leonardotrapani/hyprvoice/internal/recording"
)

func TestNewTranscriber(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid openai config",
			config: Config{
				Provider: "openai",
				APIKey:   "test-key",
				Language: "en",
				Model:    "whisper-1",
			},
			wantErr: false,
		},
		{
			name: "openai config without api key",
			config: Config{
				Provider: "openai",
				APIKey:   "",
				Language: "en",
				Model:    "whisper-1",
			},
			wantErr: true,
		},
		{
			name: "valid groq-transcription config",
			config: Config{
				Provider: "groq-transcription",
				APIKey:   "gsk-test-key",
				Language: "en",
				Model:    "whisper-large-v3",
			},
			wantErr: false,
		},
		{
			name: "groq-transcription config without api key",
			config: Config{
				Provider: "groq-transcription",
				APIKey:   "",
				Language: "en",
				Model:    "whisper-large-v3",
			},
			wantErr: true,
		},
		{
			name: "valid groq-translation config",
			config: Config{
				Provider: "groq-translation",
				APIKey:   "gsk-test-key",
				Language: "es",
				Model:    "whisper-large-v3-turbo",
			},
			wantErr: false,
		},
		{
			name: "groq-translation config without api key",
			config: Config{
				Provider: "groq-translation",
				APIKey:   "",
				Language: "es",
				Model:    "whisper-large-v3-turbo",
			},
			wantErr: true,
		},
		{
			name: "valid mistral-transcription config",
			config: Config{
				Provider: "mistral-transcription",
				APIKey:   "test-key",
				Language: "de",
				Model:    "voxtral-mini-latest",
			},
			wantErr: false,
		},
		{
			name: "mistral-transcription config without api key",
			config: Config{
				Provider: "mistral-transcription",
				APIKey:   "",
				Language: "de",
				Model:    "voxtral-mini-latest",
			},
			wantErr: true,
		},
		{
			name: "valid elevenlabs config with scribe_v1",
			config: Config{
				Provider: "elevenlabs",
				APIKey:   "test-key",
				Language: "en",
				Model:    "scribe_v1",
			},
			wantErr: false,
		},
		{
			name: "valid elevenlabs config with scribe_v2",
			config: Config{
				Provider: "elevenlabs",
				APIKey:   "test-key",
				Language: "pt",
				Model:    "scribe_v2",
			},
			wantErr: false,
		},
		{
			name: "elevenlabs config without api key",
			config: Config{
				Provider: "elevenlabs",
				APIKey:   "",
				Language: "en",
				Model:    "scribe_v1",
			},
			wantErr: true,
		},
		{
			name: "unsupported provider",
			config: Config{
				Provider: "unsupported",
				APIKey:   "test-key",
				Model:    "whisper-1",
			},
			wantErr: true,
		},
		{
			name: "empty provider",
			config: Config{
				Provider: "",
				APIKey:   "test-key",
				Model:    "whisper-1",
			},
			wantErr: true,
		},
		{
			name: "empty model uses default",
			config: Config{
				Provider: "openai",
				APIKey:   "test-key",
				Language: "en",
				Model:    "",
			},
			wantErr: false, // uses default model when empty
		},
		{
			name: "elevenlabs streaming model creates StreamingTranscriber",
			config: Config{
				Provider: "elevenlabs",
				APIKey:   "test-key",
				Language: "en",
				Model:    "scribe_v1-streaming",
			},
			wantErr: false, // streaming is now supported
		},
		{
			name: "deepgram streaming model creates StreamingTranscriber",
			config: Config{
				Provider: "deepgram",
				APIKey:   "test-key",
				Language: "en",
				Model:    "nova-3",
			},
			wantErr: false,
		},
		{
			name: "openai realtime streaming model creates StreamingTranscriber",
			config: Config{
				Provider: "openai",
				APIKey:   "test-key",
				Language: "en",
				Model:    "gpt-4o-realtime-preview",
			},
			wantErr: false,
		},
		{
			name: "unknown model returns error",
			config: Config{
				Provider: "openai",
				APIKey:   "test-key",
				Language: "en",
				Model:    "nonexistent-model",
			},
			wantErr: true,
		},
		{
			name: "valid whisper-cpp config creates adapter",
			config: Config{
				Provider: "whisper-cpp",
				Language: "en",
				Model:    "base.en",
				Threads:  4,
			},
			wantErr: false, // creates adapter even if model file doesn't exist (runtime check)
		},
		{
			name: "whisper-cpp without api key is valid",
			config: Config{
				Provider: "whisper-cpp",
				APIKey:   "", // no api key required
				Language: "en",
				Model:    "tiny.en",
			},
			wantErr: false,
		},
		{
			name: "whisper-cpp with unknown model returns error",
			config: Config{
				Provider: "whisper-cpp",
				Language: "en",
				Model:    "nonexistent-whisper-model",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transcriber, err := NewTranscriber(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTranscriber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && transcriber == nil {
				t.Errorf("NewTranscriber() returned nil transcriber")
			}
		})
	}
}

func TestConfig(t *testing.T) {
	config := Config{
		Provider: "openai",
		APIKey:   "test-key",
		Language: "en",
		Model:    "whisper-1",
	}

	if config.Provider != "openai" {
		t.Errorf("Provider mismatch: got %s, want %s", config.Provider, "openai")
	}

	if config.APIKey != "test-key" {
		t.Errorf("APIKey mismatch: got %s, want %s", config.APIKey, "test-key")
	}

	if config.Language != "en" {
		t.Errorf("Language mismatch: got %s, want %s", config.Language, "en")
	}

	if config.Model != "whisper-1" {
		t.Errorf("Model mismatch: got %s, want %s", config.Model, "whisper-1")
	}
}

// MockBatchAdapter implements BatchAdapter for testing
type MockBatchAdapter struct {
	TranscribeFunc func(ctx context.Context, audioData []byte) (string, error)
}

func (m *MockBatchAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if m.TranscribeFunc != nil {
		return m.TranscribeFunc(ctx, audioData)
	}
	return "mock transcription", nil
}

func TestSimpleTranscriber_Start(t *testing.T) {
	config := Config{
		Provider: "openai",
		APIKey:   "test-key",
		Language: "en",
		Model:    "whisper-1",
	}

	adapter := &MockBatchAdapter{}
	transcriber := NewSimpleTranscriber(config, adapter)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	frameCh := make(chan recording.AudioFrame, 10)

	// Test starting transcriber
	errCh, err := transcriber.Start(ctx, frameCh)
	if err != nil {
		t.Errorf("Start() error = %v", err)
		return
	}

	if errCh == nil {
		t.Errorf("Start() returned nil error channel")
	}

	// Test starting again should fail
	_, err = transcriber.Start(ctx, frameCh)
	if err == nil {
		t.Errorf("Start() should fail when already running")
	}

	// Stop the transcriber
	err = transcriber.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestSimpleTranscriber_Stop(t *testing.T) {
	config := Config{
		Provider: "openai",
		APIKey:   "test-key",
		Language: "en",
		Model:    "whisper-1",
	}

	adapter := &MockBatchAdapter{}
	transcriber := NewSimpleTranscriber(config, adapter)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	frameCh := make(chan recording.AudioFrame, 10)

	// Stop should be safe when not running
	err := transcriber.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error when not running = %v", err)
	}

	// Start and then stop
	_, err = transcriber.Start(ctx, frameCh)
	if err != nil {
		t.Errorf("Start() error = %v", err)
		return
	}

	// Close the frame channel to signal completion
	close(frameCh)

	err = transcriber.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Stop again should be safe
	err = transcriber.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error after already stopped = %v", err)
	}
}

func TestSimpleTranscriber_GetFinalTranscription(t *testing.T) {
	config := Config{
		Provider: "openai",
		APIKey:   "test-key",
		Language: "en",
		Model:    "whisper-1",
	}

	adapter := &MockBatchAdapter{
		TranscribeFunc: func(ctx context.Context, audioData []byte) (string, error) {
			return "test transcription", nil
		},
	}
	transcriber := NewSimpleTranscriber(config, adapter)

	// Test getting transcription before any processing
	transcription, err := transcriber.GetFinalTranscription()
	if err != nil {
		t.Errorf("GetFinalTranscription() error = %v", err)
		return
	}

	// Should return empty string initially
	if transcription != "" {
		t.Errorf("GetFinalTranscription() = %q, want empty string", transcription)
	}
}

func TestSimpleTranscriber_CollectAudio(t *testing.T) {
	config := Config{
		Provider: "openai",
		APIKey:   "test-key",
		Language: "en",
		Model:    "whisper-1",
	}

	adapter := &MockBatchAdapter{}
	transcriber := NewSimpleTranscriber(config, adapter)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	frameCh := make(chan recording.AudioFrame, 10)
	errCh := make(chan error, 1)

	// Start collecting audio in background
	transcriber.wg.Add(1)
	go transcriber.collectAudio(ctx, frameCh, errCh)

	// Send some test audio frames
	testData1 := []byte{1, 2, 3, 4}
	testData2 := []byte{5, 6, 7, 8}

	frame1 := recording.AudioFrame{
		Data:      testData1,
		Timestamp: time.Now(),
	}

	frame2 := recording.AudioFrame{
		Data:      testData2,
		Timestamp: time.Now(),
	}

	frameCh <- frame1
	frameCh <- frame2
	close(frameCh)

	// Wait for processing to complete
	transcriber.wg.Wait()

	// Check that audio was collected
	if len(transcriber.audioBuffer) != len(testData1)+len(testData2) {
		t.Errorf("Audio buffer length = %d, want %d", len(transcriber.audioBuffer), len(testData1)+len(testData2))
	}
}

func TestSimpleTranscriber_TranscribeAll(t *testing.T) {
	tests := []struct {
		name           string
		audioData      []byte
		mockResult     string
		mockError      error
		expectError    bool
		expectedResult string
	}{
		{
			name:           "successful transcription",
			audioData:      []byte{1, 2, 3, 4},
			mockResult:     "hello world",
			mockError:      nil,
			expectError:    false,
			expectedResult: "hello world",
		},
		{
			name:           "empty audio data",
			audioData:      []byte{},
			mockResult:     "",
			mockError:      nil,
			expectError:    false,
			expectedResult: "",
		},
		{
			name:           "transcription error",
			audioData:      []byte{1, 2, 3, 4},
			mockResult:     "",
			mockError:      fmt.Errorf("api error"),
			expectError:    true,
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Provider: "openai",
				APIKey:   "test-key",
				Language: "en",
				Model:    "whisper-1",
			}

			adapter := &MockBatchAdapter{
				TranscribeFunc: func(ctx context.Context, audioData []byte) (string, error) {
					return tt.mockResult, tt.mockError
				},
			}
			transcriber := NewSimpleTranscriber(config, adapter)

			// Set up audio buffer
			transcriber.audioBuffer = tt.audioData

			ctx := context.Background()
			err := transcriber.transcribeAll(ctx)

			if (err != nil) != tt.expectError {
				t.Errorf("transcribeAll() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				result, err := transcriber.GetFinalTranscription()
				if err != nil {
					t.Errorf("GetFinalTranscription() error = %v", err)
					return
				}

				if result != tt.expectedResult {
					t.Errorf("GetFinalTranscription() = %q, want %q", result, tt.expectedResult)
				}
			}
		})
	}
}

func TestNewSimpleTranscriber(t *testing.T) {
	config := Config{
		Provider: "openai",
		APIKey:   "test-key",
		Language: "en",
		Model:    "whisper-1",
	}

	adapter := &MockBatchAdapter{}
	transcriber := NewSimpleTranscriber(config, adapter)

	if transcriber == nil {
		t.Errorf("NewSimpleTranscriber() returned nil")
		return
	}

	if transcriber.adapter != adapter {
		t.Errorf("Adapter not set correctly")
	}

	if transcriber.config.Provider != config.Provider {
		t.Errorf("Config not set correctly")
	}

	if transcriber.running {
		t.Errorf("Transcriber should not be running initially")
	}

	if len(transcriber.audioBuffer) != 0 {
		t.Errorf("Audio buffer should be empty initially")
	}
}

func TestTranscriptionAdapter(t *testing.T) {
	adapter := &MockBatchAdapter{
		TranscribeFunc: func(ctx context.Context, audioData []byte) (string, error) {
			return "test result", nil
		},
	}

	ctx := context.Background()
	audioData := []byte{1, 2, 3, 4}

	result, err := adapter.Transcribe(ctx, audioData)
	if err != nil {
		t.Errorf("Transcribe() error = %v", err)
		return
	}

	if result != "test result" {
		t.Errorf("Transcribe() = %q, want %q", result, "test result")
	}
}

func TestOpenAIAdapter_Creation(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     *provider.EndpointConfig
		apiKey       string
		model        string
		language     string
		keywords     []string
		providerName string
	}{
		{
			name:         "openai with nil endpoint uses default",
			endpoint:     nil,
			apiKey:       "sk-test-key",
			model:        "whisper-1",
			language:     "en",
			keywords:     []string{"hello", "world"},
			providerName: "openai",
		},
		{
			name:         "openai with explicit endpoint",
			endpoint:     &provider.EndpointConfig{BaseURL: "https://api.openai.com", Path: "/v1/audio/transcriptions"},
			apiKey:       "sk-test-key",
			model:        "whisper-1",
			language:     "es",
			keywords:     nil,
			providerName: "openai",
		},
		{
			name:         "groq with custom endpoint",
			endpoint:     &provider.EndpointConfig{BaseURL: "https://api.groq.com/openai", Path: "/v1/audio/transcriptions"},
			apiKey:       "gsk-test-key",
			model:        "whisper-large-v3",
			language:     "fr",
			keywords:     []string{"bonjour"},
			providerName: "groq",
		},
		{
			name:         "mistral with custom endpoint",
			endpoint:     &provider.EndpointConfig{BaseURL: "https://api.mistral.ai", Path: "/v1/audio/transcriptions"},
			apiKey:       "mistral-test-key",
			model:        "voxtral-mini-latest",
			language:     "de",
			keywords:     nil,
			providerName: "mistral",
		},
		{
			name:         "auto language",
			endpoint:     nil,
			apiKey:       "sk-test-key",
			model:        "whisper-1",
			language:     "", // auto
			keywords:     nil,
			providerName: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewOpenAIAdapter(tt.endpoint, tt.apiKey, tt.model, tt.language, tt.keywords, tt.providerName)
			if adapter == nil {
				t.Errorf("NewOpenAIAdapter() returned nil")
				return
			}

			if adapter.model != tt.model {
				t.Errorf("model = %q, want %q", adapter.model, tt.model)
			}

			if adapter.language != tt.language {
				t.Errorf("language = %q, want %q", adapter.language, tt.language)
			}

			if adapter.providerName != tt.providerName {
				t.Errorf("providerName = %q, want %q", adapter.providerName, tt.providerName)
			}

			if len(adapter.keywords) != len(tt.keywords) {
				t.Errorf("keywords len = %d, want %d", len(adapter.keywords), len(tt.keywords))
			}
		})
	}
}

func TestOpenAIAdapterFromConfig(t *testing.T) {
	config := Config{
		Provider: "openai",
		APIKey:   "sk-test-key",
		Model:    "whisper-1",
		Language: "en",
		Keywords: []string{"test"},
	}

	adapter := NewOpenAIAdapterFromConfig(config)
	if adapter == nil {
		t.Errorf("NewOpenAIAdapterFromConfig() returned nil")
		return
	}

	if adapter.model != config.Model {
		t.Errorf("model = %q, want %q", adapter.model, config.Model)
	}

	if adapter.language != config.Language {
		t.Errorf("language = %q, want %q", adapter.language, config.Language)
	}

	if adapter.providerName != "openai" {
		t.Errorf("providerName = %q, want %q", adapter.providerName, "openai")
	}
}

// MockStreamingAdapter implements StreamingAdapter for testing
type MockStreamingAdapter struct {
	StartFunc     func(ctx context.Context, language string) error
	SendChunkFunc func(audio []byte) error
	ResultsFunc   func() <-chan TranscriptionResult
	CloseFunc     func() error

	resultsCh chan TranscriptionResult
}

func NewMockStreamingAdapter() *MockStreamingAdapter {
	return &MockStreamingAdapter{
		resultsCh: make(chan TranscriptionResult, 10),
	}
}

func (m *MockStreamingAdapter) Start(ctx context.Context, language string) error {
	if m.StartFunc != nil {
		return m.StartFunc(ctx, language)
	}
	return nil
}

func (m *MockStreamingAdapter) SendChunk(audio []byte) error {
	if m.SendChunkFunc != nil {
		return m.SendChunkFunc(audio)
	}
	return nil
}

func (m *MockStreamingAdapter) Results() <-chan TranscriptionResult {
	if m.ResultsFunc != nil {
		return m.ResultsFunc()
	}
	return m.resultsCh
}

func (m *MockStreamingAdapter) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	close(m.resultsCh)
	return nil
}

func (m *MockStreamingAdapter) SendResult(result TranscriptionResult) {
	m.resultsCh <- result
}

func TestStreamingTranscriber_Start(t *testing.T) {
	adapter := NewMockStreamingAdapter()
	transcriber := NewStreamingTranscriber(adapter, "en")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	frameCh := make(chan recording.AudioFrame, 10)

	errCh, err := transcriber.Start(ctx, frameCh)
	if err != nil {
		t.Errorf("Start() error = %v", err)
		return
	}

	if errCh == nil {
		t.Errorf("Start() returned nil error channel")
	}

	close(frameCh)
	err = transcriber.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestStreamingTranscriber_StartError(t *testing.T) {
	adapter := NewMockStreamingAdapter()
	adapter.StartFunc = func(ctx context.Context, language string) error {
		return fmt.Errorf("connection failed")
	}

	transcriber := NewStreamingTranscriber(adapter, "en")

	ctx := context.Background()
	frameCh := make(chan recording.AudioFrame, 10)

	_, err := transcriber.Start(ctx, frameCh)
	if err == nil {
		t.Errorf("Start() should fail when adapter.Start fails")
	}
}

func TestStreamingTranscriber_AccumulatesResults(t *testing.T) {
	adapter := NewMockStreamingAdapter()
	transcriber := NewStreamingTranscriber(adapter, "en")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	frameCh := make(chan recording.AudioFrame, 10)

	_, err := transcriber.Start(ctx, frameCh)
	if err != nil {
		t.Errorf("Start() error = %v", err)
		return
	}

	// send some final results
	adapter.SendResult(TranscriptionResult{Text: "hello", IsFinal: true})
	adapter.SendResult(TranscriptionResult{Text: "world", IsFinal: true})

	// give time for results to be processed
	time.Sleep(50 * time.Millisecond)

	close(frameCh)
	err = transcriber.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	result, err := transcriber.GetFinalTranscription()
	if err != nil {
		t.Errorf("GetFinalTranscription() error = %v", err)
		return
	}

	if result != "hello world" {
		t.Errorf("GetFinalTranscription() = %q, want %q", result, "hello world")
	}
}

func TestStreamingTranscriber_IgnoresPartialResults(t *testing.T) {
	adapter := NewMockStreamingAdapter()
	transcriber := NewStreamingTranscriber(adapter, "en")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	frameCh := make(chan recording.AudioFrame, 10)

	_, err := transcriber.Start(ctx, frameCh)
	if err != nil {
		t.Errorf("Start() error = %v", err)
		return
	}

	// partial results should be ignored
	adapter.SendResult(TranscriptionResult{Text: "hel", IsFinal: false})
	adapter.SendResult(TranscriptionResult{Text: "hello", IsFinal: true})
	adapter.SendResult(TranscriptionResult{Text: "hello wor", IsFinal: false})

	time.Sleep(50 * time.Millisecond)

	close(frameCh)
	err = transcriber.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	result, err := transcriber.GetFinalTranscription()
	if err != nil {
		t.Errorf("GetFinalTranscription() error = %v", err)
		return
	}

	if result != "hello" {
		t.Errorf("GetFinalTranscription() = %q, want %q", result, "hello")
	}
}

func TestStreamingTranscriber_HandlesErrors(t *testing.T) {
	adapter := NewMockStreamingAdapter()
	transcriber := NewStreamingTranscriber(adapter, "en")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	frameCh := make(chan recording.AudioFrame, 10)

	errCh, err := transcriber.Start(ctx, frameCh)
	if err != nil {
		t.Errorf("Start() error = %v", err)
		return
	}

	// send an error result
	adapter.SendResult(TranscriptionResult{Error: fmt.Errorf("transcription error")})

	// error should be received on errCh
	select {
	case e := <-errCh:
		if e == nil {
			t.Errorf("expected error on errCh")
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("timeout waiting for error on errCh")
	}

	close(frameCh)
	_ = transcriber.Stop(ctx)
}

func TestStreamingTranscriber_SendsAudioChunks(t *testing.T) {
	var receivedChunks [][]byte
	adapter := NewMockStreamingAdapter()
	adapter.SendChunkFunc = func(audio []byte) error {
		chunk := make([]byte, len(audio))
		copy(chunk, audio)
		receivedChunks = append(receivedChunks, chunk)
		return nil
	}

	transcriber := NewStreamingTranscriber(adapter, "en")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	frameCh := make(chan recording.AudioFrame, 10)

	_, err := transcriber.Start(ctx, frameCh)
	if err != nil {
		t.Errorf("Start() error = %v", err)
		return
	}

	// send audio frames
	frameCh <- recording.AudioFrame{Data: []byte{1, 2, 3, 4}}
	frameCh <- recording.AudioFrame{Data: []byte{5, 6, 7, 8}}

	time.Sleep(50 * time.Millisecond)

	close(frameCh)
	err = transcriber.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	if len(receivedChunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(receivedChunks))
	}
}

func TestStreamingTranscriber_ContextCancellation(t *testing.T) {
	adapter := NewMockStreamingAdapter()
	transcriber := NewStreamingTranscriber(adapter, "en")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	frameCh := make(chan recording.AudioFrame, 10)

	_, err := transcriber.Start(ctx, frameCh)
	if err != nil {
		t.Errorf("Start() error = %v", err)
		return
	}

	// cancel context
	cancel()

	// stop should complete without hanging
	done := make(chan struct{})
	go func() {
		_ = transcriber.Stop(context.Background())
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Errorf("Stop() timed out after context cancellation")
	}
}

func TestStreamingTranscriber_GetFinalTranscriptionSafe(t *testing.T) {
	adapter := NewMockStreamingAdapter()
	transcriber := NewStreamingTranscriber(adapter, "en")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	frameCh := make(chan recording.AudioFrame, 10)

	_, err := transcriber.Start(ctx, frameCh)
	if err != nil {
		t.Errorf("Start() error = %v", err)
		return
	}

	// call GetFinalTranscription concurrently while results are being added
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			_, _ = transcriber.GetFinalTranscription()
			time.Sleep(time.Millisecond)
		}
		close(done)
	}()

	// send results concurrently
	for i := 0; i < 10; i++ {
		adapter.SendResult(TranscriptionResult{Text: "word", IsFinal: true})
		time.Sleep(5 * time.Millisecond)
	}

	<-done

	close(frameCh)
	err = transcriber.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestNewTranscriber_LanguageFallback(t *testing.T) {
	// test that incompatible language falls back to auto-detect (no error)
	// distil-whisper-large-v3-en only supports English
	config := Config{
		Provider: "groq-transcription",
		APIKey:   "test-key",
		Language: "es", // Spanish not supported by English-only model
		Model:    "distil-whisper-large-v3-en",
	}

	// should succeed (fallback to auto), not error
	transcriber, err := NewTranscriber(config)
	if err != nil {
		t.Errorf("NewTranscriber() should fall back to auto, got error: %v", err)
		return
	}

	if transcriber == nil {
		t.Errorf("NewTranscriber() returned nil transcriber")
	}
}

func TestNewTranscriber_AutoLanguageNoFallback(t *testing.T) {
	// test that auto language never triggers warning/fallback
	config := Config{
		Provider: "groq-transcription",
		APIKey:   "test-key",
		Language: "", // auto
		Model:    "distil-whisper-large-v3-en",
	}

	transcriber, err := NewTranscriber(config)
	if err != nil {
		t.Errorf("NewTranscriber() error = %v", err)
		return
	}

	if transcriber == nil {
		t.Errorf("NewTranscriber() returned nil transcriber")
	}
}

func TestNewTranscriber_CompatibleLanguageNoFallback(t *testing.T) {
	// test that compatible language works normally
	config := Config{
		Provider: "groq-transcription",
		APIKey:   "test-key",
		Language: "en", // English supported by English-only model
		Model:    "distil-whisper-large-v3-en",
	}

	transcriber, err := NewTranscriber(config)
	if err != nil {
		t.Errorf("NewTranscriber() error = %v", err)
		return
	}

	if transcriber == nil {
		t.Errorf("NewTranscriber() returned nil transcriber")
	}
}

func TestNewTranscriber_MultilingualModelAllLanguages(t *testing.T) {
	// test that multilingual model accepts any language without fallback
	config := Config{
		Provider: "groq-transcription",
		APIKey:   "test-key",
		Language: "es",               // Spanish
		Model:    "whisper-large-v3", // multilingual
	}

	transcriber, err := NewTranscriber(config)
	if err != nil {
		t.Errorf("NewTranscriber() error = %v", err)
		return
	}

	if transcriber == nil {
		t.Errorf("NewTranscriber() returned nil transcriber")
	}
}
