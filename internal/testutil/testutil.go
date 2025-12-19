package testutil

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/recording"
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
			APIKey:   "test-api-key",
			Language: "",
			Model:    "whisper-1",
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
			APIKey:   "", // Invalid
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

// MockTranscriberAdapter implements transcriber.TranscriptionAdapter for testing
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
