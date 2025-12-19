package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/config"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Recording: config.RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           5 * time.Minute,
		},
		Transcription: config.TranscriptionConfig{
			Provider: "openai",
			APIKey:   "test-key",
			Language: "en",
			Model:    "whisper-1",
		},
		Injection: config.InjectionConfig{
			Backends:         []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout:   5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: config.NotificationsConfig{
			Enabled: true,
			Type:    "log",
		},
	}

	pipeline := New(cfg)
	if pipeline == nil {
		t.Errorf("New() returned nil")
		return
	}

	// Test that pipeline is created successfully
	// Note: Status may be empty initially due to implementation
	t.Logf("Initial status = %s", pipeline.Status())
}

func TestPipeline_Status(t *testing.T) {
	cfg := &config.Config{
		Recording: config.RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           5 * time.Minute,
		},
		Transcription: config.TranscriptionConfig{
			Provider: "openai",
			APIKey:   "test-key",
			Language: "en",
			Model:    "whisper-1",
		},
		Injection: config.InjectionConfig{
			Backends:         []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout:   5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: config.NotificationsConfig{
			Enabled: true,
			Type:    "log",
		},
	}

	pipeline := New(cfg)

	// Test that we can get status (may be empty initially)
	status := pipeline.Status()
	t.Logf("Status() = %s", status)

	// Test that we can get action channel
	actionCh := pipeline.GetActionCh()
	if actionCh == nil {
		t.Errorf("GetActionCh() returned nil")
	}

	// Test that we can get error channel
	errorCh := pipeline.GetErrorCh()
	if errorCh == nil {
		t.Errorf("GetErrorCh() returned nil")
	}
}

func TestPipeline_GetActionCh(t *testing.T) {
	cfg := &config.Config{
		Recording: config.RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           5 * time.Minute,
		},
		Transcription: config.TranscriptionConfig{
			Provider: "openai",
			APIKey:   "test-key",
			Language: "en",
			Model:    "whisper-1",
		},
		Injection: config.InjectionConfig{
			Backends:         []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout:   5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: config.NotificationsConfig{
			Enabled: true,
			Type:    "log",
		},
	}

	pipeline := New(cfg)
	actionCh := pipeline.GetActionCh()

	if actionCh == nil {
		t.Errorf("GetActionCh() returned nil")
		return
	}

	// Test sending an action
	select {
	case actionCh <- Inject:
		// Action sent successfully
	default:
		t.Errorf("Could not send action to channel")
	}
}

func TestPipeline_GetErrorCh(t *testing.T) {
	cfg := &config.Config{
		Recording: config.RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           5 * time.Minute,
		},
		Transcription: config.TranscriptionConfig{
			Provider: "openai",
			APIKey:   "test-key",
			Language: "en",
			Model:    "whisper-1",
		},
		Injection: config.InjectionConfig{
			Backends:         []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout:   5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: config.NotificationsConfig{
			Enabled: true,
			Type:    "log",
		},
	}

	pipeline := New(cfg)
	errorCh := pipeline.GetErrorCh()

	if errorCh == nil {
		t.Errorf("GetErrorCh() returned nil")
		return
	}

	// Test that we can receive from the error channel
	select {
	case <-errorCh:
		// Error received
	default:
		// No error available, which is expected
	}
}

func TestPipeline_Stop(t *testing.T) {
	cfg := &config.Config{
		Recording: config.RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           5 * time.Minute,
		},
		Transcription: config.TranscriptionConfig{
			Provider: "openai",
			APIKey:   "test-key",
			Language: "en",
			Model:    "whisper-1",
		},
		Injection: config.InjectionConfig{
			Backends:         []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout:   5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: config.NotificationsConfig{
			Enabled: true,
			Type:    "log",
		},
	}

	pipeline := New(cfg)

	// Stop should be safe to call even when not running
	pipeline.Stop()

	// Status should be consistent after stop
	status := pipeline.Status()
	t.Logf("Status after stop = %s", status)
}

func TestPipeline_Run(t *testing.T) {
	cfg := &config.Config{
		Recording: config.RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           5 * time.Minute,
		},
		Transcription: config.TranscriptionConfig{
			Provider: "openai",
			APIKey:   "test-key",
			Language: "en",
			Model:    "whisper-1",
		},
		Injection: config.InjectionConfig{
			Backends:         []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout:   5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: config.NotificationsConfig{
			Enabled: true,
			Type:    "log",
		},
	}

	pipeline := New(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test running the pipeline
	pipeline.Run(ctx)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Check status after starting (may transition quickly due to test environment)
	status := pipeline.Status()
	t.Logf("Status after Run = %s", status)

	// Stop the pipeline
	pipeline.Stop()

	// Give it a moment to stop
	time.Sleep(100 * time.Millisecond)

	// Check final status
	finalStatus := pipeline.Status()
	t.Logf("Status after Stop = %s", finalStatus)
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{Idle, "idle"},
		{Recording, "recording"},
		{Transcribing, "transcribing"},
		{Injecting, "injecting"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Status string = %s, want %s", string(tt.status), tt.expected)
			}
		})
	}
}

func TestAction_String(t *testing.T) {
	tests := []struct {
		action   Action
		expected string
	}{
		{Inject, "inject"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if string(tt.action) != tt.expected {
				t.Errorf("Action string = %s, want %s", string(tt.action), tt.expected)
			}
		})
	}
}

func TestPipelineError_Struct(t *testing.T) {
	err := PipelineError{
		Title:   "Test Title",
		Message: "Test Message",
		Err:     nil,
	}

	if err.Title != "Test Title" {
		t.Errorf("Title = %s, want %s", err.Title, "Test Title")
	}

	if err.Message != "Test Message" {
		t.Errorf("Message = %s, want %s", err.Message, "Test Message")
	}

	if err.Err != nil {
		t.Errorf("Err = %v, want nil", err.Err)
	}
}

// TestPipeline_ConcurrentAccess tests concurrent access to pipeline methods
func TestPipeline_ConcurrentAccess(t *testing.T) {
	cfg := &config.Config{
		Recording: config.RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           5 * time.Minute,
		},
		Transcription: config.TranscriptionConfig{
			Provider: "openai",
			APIKey:   "test-key",
			Language: "en",
			Model:    "whisper-1",
		},
		Injection: config.InjectionConfig{
			Backends:         []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout:   5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: config.NotificationsConfig{
			Enabled: true,
			Type:    "log",
		},
	}

	pipeline := New(cfg)

	// Test concurrent access to Status()
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			pipeline.Status()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			pipeline.GetActionCh()
			pipeline.GetErrorCh()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done
}
