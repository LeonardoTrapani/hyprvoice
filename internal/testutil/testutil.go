package testutil

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/injection"
	"github.com/leonardotrapani/hyprvoice/internal/llm"
	"github.com/leonardotrapani/hyprvoice/internal/recording"
	"github.com/leonardotrapani/hyprvoice/internal/transcriber"
)

// TestConfig returns a valid configuration for testing
func TestConfig() *config.Config {
	return &config.Config{
		Recording: config.RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			Device:            "",
			ChannelBufferSize: 30,
			Timeout:           5 * time.Minute,
		},
		Transcription: config.TranscriptionConfig{
			Provider: "openai",
			Language: "",
			Model:    "whisper-1",
		},
		Providers: map[string]config.ProviderConfig{
			"openai": {APIKey: "test-api-key"},
		},
		Injection: config.InjectionConfig{
			Backends:         []string{"ydotool", "wtype", "clipboard"},
			YdotoolTimeout:   5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: config.NotificationsConfig{
			Enabled: true,
			Type:    "log",
		},
	}
}

// TestConfigWithInvalidValues returns a config with invalid values for testing validation
func TestConfigWithInvalidValues() *config.Config {
	return &config.Config{
		Recording: config.RecordingConfig{
			SampleRate:        0,  // Invalid
			Channels:          0,  // Invalid
			Format:            "", // Invalid
			BufferSize:        0,  // Invalid
			ChannelBufferSize: 0,  // Invalid
			Timeout:           0,  // Invalid
		},
		Transcription: config.TranscriptionConfig{
			Provider: "", // Invalid
			Model:    "", // Invalid
		},
		Injection: config.InjectionConfig{
			Backends:         []string{}, // Invalid (empty)
			YdotoolTimeout:   0,          // Invalid
			WtypeTimeout:     0,          // Invalid
			ClipboardTimeout: 0,          // Invalid
		},
		Notifications: config.NotificationsConfig{
			Type: "invalid", // Invalid
		},
	}
}

// CreateTempConfigFile creates a temporary config file for testing
func CreateTempConfigFile(t *testing.T, configContent string) string {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}

	return configPath
}

// MockCommandExecutor provides a way to mock exec.Command calls
type MockCommandExecutor struct {
	Commands []MockCommand
}

type MockCommand struct {
	Command string
	Args    []string
	Output  string
	Error   error
}

func (m *MockCommandExecutor) AddCommand(cmd string, args []string, output string, err error) {
	m.Commands = append(m.Commands, MockCommand{
		Command: cmd,
		Args:    args,
		Output:  output,
		Error:   err,
	})
}

// MockAudioFrame creates a test audio frame
func MockAudioFrame(data []byte) recording.AudioFrame {
	if data == nil {
		data = make([]byte, 1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
	}

	return recording.AudioFrame{
		Data:      data,
		Timestamp: time.Now(),
	}
}

// MockTranscriberAdapter implements transcriber.BatchAdapter for testing
type MockTranscriberAdapter struct {
	TranscribeFunc func(ctx context.Context, audioData []byte) (string, error)
}

func (m *MockTranscriberAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if m.TranscribeFunc != nil {
		return m.TranscribeFunc(ctx, audioData)
	}
	return "mock transcription", nil
}

// NewMockTranscriberAdapter creates a mock transcriber adapter
func NewMockTranscriberAdapter() *MockTranscriberAdapter {
	return &MockTranscriberAdapter{}
}

// TestContext returns a context with timeout for testing
func TestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// WaitForCondition waits for a condition to be true or times out
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Condition not met within %v", timeout)
		default:
			if condition() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// CaptureOutput captures stdout/stderr for testing
func CaptureOutput(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	return string(out)
}

// MockRecorder implements recording.Recorder for testing
type MockRecorder struct {
	Frames     []recording.AudioFrame
	StartError error

	mu        sync.Mutex
	recording atomic.Bool
	stopCh    chan struct{}
}

func NewMockRecorder() *MockRecorder {
	return &MockRecorder{
		Frames: []recording.AudioFrame{MockAudioFrame(nil)},
	}
}

func (m *MockRecorder) Start(ctx context.Context) (<-chan recording.AudioFrame, <-chan error, error) {
	if m.StartError != nil {
		return nil, nil, m.StartError
	}

	m.mu.Lock()
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	m.recording.Store(true)

	frameCh := make(chan recording.AudioFrame, len(m.Frames)+1)
	errCh := make(chan error, 1)

	go func() {
		defer close(frameCh)
		defer close(errCh)

		for _, frame := range m.Frames {
			select {
			case <-ctx.Done():
				return
			case <-m.stopCh:
				return
			case frameCh <- frame:
			}
		}

		// keep channel open until stopped
		select {
		case <-ctx.Done():
		case <-m.stopCh:
		}
	}()

	return frameCh, errCh, nil
}

func (m *MockRecorder) Stop() {
	if !m.recording.Load() {
		return
	}
	m.recording.Store(false)

	m.mu.Lock()
	if m.stopCh != nil {
		close(m.stopCh)
		m.stopCh = nil
	}
	m.mu.Unlock()
}

func (m *MockRecorder) IsRecording() bool {
	return m.recording.Load()
}

// MockTranscriber implements transcriber.Transcriber for testing
type MockTranscriber struct {
	Transcription string
	StartError    error
	StopError     error
	GetError      error

	mu      sync.Mutex
	started bool
}

func NewMockTranscriber(transcription string) *MockTranscriber {
	return &MockTranscriber{Transcription: transcription}
}

func (m *MockTranscriber) Start(ctx context.Context, frameCh <-chan recording.AudioFrame) (<-chan error, error) {
	if m.StartError != nil {
		return nil, m.StartError
	}

	m.mu.Lock()
	m.started = true
	m.mu.Unlock()

	errCh := make(chan error, 1)

	// drain frames in background
	go func() {
		defer close(errCh)
		for range frameCh {
		}
	}()

	return errCh, nil
}

func (m *MockTranscriber) Stop(ctx context.Context) error {
	m.mu.Lock()
	m.started = false
	m.mu.Unlock()
	return m.StopError
}

func (m *MockTranscriber) GetFinalTranscription() (string, error) {
	if m.GetError != nil {
		return "", m.GetError
	}
	return m.Transcription, nil
}

// MockInjector implements injection.Injector for testing
type MockInjector struct {
	InjectedTexts []string
	InjectError   error

	mu sync.Mutex
}

func NewMockInjector() *MockInjector {
	return &MockInjector{}
}

func (m *MockInjector) Inject(ctx context.Context, text string) error {
	if m.InjectError != nil {
		return m.InjectError
	}
	m.mu.Lock()
	m.InjectedTexts = append(m.InjectedTexts, text)
	m.mu.Unlock()
	return nil
}

func (m *MockInjector) GetInjectedTexts() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.InjectedTexts))
	copy(result, m.InjectedTexts)
	return result
}

// MockLLMAdapter implements llm.Adapter for testing
type MockLLMAdapter struct {
	ProcessedText string
	ProcessError  error

	mu            sync.Mutex
	ProcessCalled bool
	InputText     string
}

func NewMockLLMAdapter(processedText string) *MockLLMAdapter {
	return &MockLLMAdapter{ProcessedText: processedText}
}

func (m *MockLLMAdapter) Process(ctx context.Context, text string) (string, error) {
	m.mu.Lock()
	m.ProcessCalled = true
	m.InputText = text
	m.mu.Unlock()

	if m.ProcessError != nil {
		return "", m.ProcessError
	}
	return m.ProcessedText, nil
}

// Factory helpers for pipeline testing

// MockRecorderFactory returns a factory that creates the given mock recorder
func MockRecorderFactory(mock *MockRecorder) func(cfg recording.Config) recording.Recorder {
	return func(cfg recording.Config) recording.Recorder {
		return mock
	}
}

// MockTranscriberFactory returns a factory that creates the given mock transcriber
func MockTranscriberFactory(mock *MockTranscriber) func(cfg transcriber.Config) (transcriber.Transcriber, error) {
	return func(cfg transcriber.Config) (transcriber.Transcriber, error) {
		return mock, nil
	}
}

// MockInjectorFactory returns a factory that creates the given mock injector
func MockInjectorFactory(mock *MockInjector) func(cfg injection.Config) injection.Injector {
	return func(cfg injection.Config) injection.Injector {
		return mock
	}
}

// MockLLMAdapterFactory returns a factory that creates the given mock LLM adapter
func MockLLMAdapterFactory(mock *MockLLMAdapter) func(cfg llm.Config) (llm.Adapter, error) {
	return func(cfg llm.Config) (llm.Adapter, error) {
		return mock, nil
	}
}
