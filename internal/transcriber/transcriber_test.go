package transcriber

import (
	"context"
	"fmt"
	"testing"
	"time"

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
			name: "empty model",
			config: Config{
				Provider: "openai",
				APIKey:   "test-key",
				Language: "en",
				Model:    "",
			},
			wantErr: false, // Model validation is not implemented in NewTranscriber
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

// MockTranscriptionAdapter implements TranscriptionAdapter for testing
type MockTranscriptionAdapter struct {
	TranscribeFunc func(ctx context.Context, audioData []byte) (string, error)
}

func (m *MockTranscriptionAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
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

	adapter := &MockTranscriptionAdapter{}
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

	adapter := &MockTranscriptionAdapter{}
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

	adapter := &MockTranscriptionAdapter{
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

	adapter := &MockTranscriptionAdapter{}
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

			adapter := &MockTranscriptionAdapter{
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

	adapter := &MockTranscriptionAdapter{}
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
	adapter := &MockTranscriptionAdapter{
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
