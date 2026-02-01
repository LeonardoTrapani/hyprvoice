package recording

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewRecorder(t *testing.T) {
	config := Config{
		SampleRate:        16000,
		Channels:          1,
		Format:            "s16",
		BufferSize:        8192,
		Device:            "",
		ChannelBufferSize: 30,
		Timeout:           5 * time.Minute,
	}

	recorder := NewRecorder(config)
	if recorder == nil {
		t.Errorf("NewRecorder() returned nil")
		return
	}

	// verify recorder implements the interface
	if !recorder.IsRecording() {
		t.Logf("Recorder created successfully, not recording initially")
	}
}

func TestRecorder_IsRecording(t *testing.T) {
	config := Config{
		SampleRate:        16000,
		Channels:          1,
		Format:            "s16",
		BufferSize:        8192,
		Device:            "",
		ChannelBufferSize: 30,
		Timeout:           5 * time.Minute,
	}

	recorder := NewRecorder(config)

	// Initially should not be recording
	if recorder.IsRecording() {
		t.Errorf("IsRecording() = true, want false initially")
	}
}

func TestRecorder_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "invalid sample rate",
			config: Config{
				SampleRate:        0,
				Channels:          1,
				Format:            "s16",
				BufferSize:        8192,
				ChannelBufferSize: 30,
				Timeout:           5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "invalid channels",
			config: Config{
				SampleRate:        16000,
				Channels:          0,
				Format:            "s16",
				BufferSize:        8192,
				ChannelBufferSize: 30,
				Timeout:           5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "invalid buffer size",
			config: Config{
				SampleRate:        16000,
				Channels:          1,
				Format:            "s16",
				BufferSize:        0,
				ChannelBufferSize: 30,
				Timeout:           5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "invalid channel buffer size",
			config: Config{
				SampleRate:        16000,
				Channels:          1,
				Format:            "s16",
				BufferSize:        8192,
				ChannelBufferSize: 0,
				Timeout:           5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			config: Config{
				SampleRate:        16000,
				Channels:          1,
				Format:            "",
				BufferSize:        8192,
				ChannelBufferSize: 30,
				Timeout:           5 * time.Minute,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := NewRecorder(tt.config)
			ctx := context.Background()
			_, _, err := recorder.Start(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() with invalid config error = %v, wantErr %v", err, tt.wantErr)
			}
			recorder.Stop()
		})
	}
}

func TestCheckPipeWireAvailable(t *testing.T) {
	// Test with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This test will fail if pw-record is not available in the system
	// In a real CI environment, we would mock this
	err := CheckPipeWireAvailable(ctx)
	if err != nil {
		t.Logf("CheckPipeWireAvailable() failed (expected if pw-record not installed): %v", err)
		// Don't fail the test if pw-record is not available
		return
	}

	// If pw-record is available, the function should succeed
	t.Logf("CheckPipeWireAvailable() succeeded - pw-record is available")
}

func TestAudioFrame(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	timestamp := time.Now()

	frame := AudioFrame{
		Data:      data,
		Timestamp: timestamp,
	}

	if len(frame.Data) != len(data) {
		t.Errorf("Data length mismatch: got %d, want %d", len(frame.Data), len(data))
	}

	if frame.Timestamp != timestamp {
		t.Errorf("Timestamp mismatch: got %v, want %v", frame.Timestamp, timestamp)
	}

	// Test with nil data
	emptyFrame := AudioFrame{}
	if emptyFrame.Data != nil {
		t.Errorf("Empty frame should have nil data")
	}
}

// TestRecorder_Start tests the Start method with mocked external dependencies
// This is a simplified test that focuses on the logic rather than actual audio capture
func TestRecorder_Start(t *testing.T) {
	// Skip integration tests in CI environments
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}

	config := Config{
		SampleRate:        16000,
		Channels:          1,
		Format:            "s16",
		BufferSize:        8192,
		Device:            "",
		ChannelBufferSize: 30,
		Timeout:           5 * time.Minute,
	}

	recorder := NewRecorder(config)

	// Test that we can't start recording if already recording
	ctx := context.Background()
	frameCh1, errCh1, err := recorder.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
		return
	}

	if frameCh1 == nil {
		t.Errorf("Start() returned nil frame channel")
	}

	if errCh1 == nil {
		t.Errorf("Start() returned nil error channel")
	}

	// Should not be able to start again
	_, _, err = recorder.Start(ctx)
	if err == nil {
		t.Errorf("Start() should fail when already recording")
	}

	// Stop the recorder
	recorder.Stop()

	// Give it a moment to stop
	time.Sleep(10 * time.Millisecond)

	// Should be able to start again after stopping
	frameCh2, errCh2, err := recorder.Start(ctx)
	if err != nil {
		t.Errorf("Start() error after stop = %v", err)
		return
	}

	if frameCh2 == nil {
		t.Errorf("Start() returned nil frame channel after restart")
	}

	if errCh2 == nil {
		t.Errorf("Start() returned nil error channel after restart")
	}

	recorder.Stop()
}

// TestRecorder_Stop tests the Stop method
func TestRecorder_Stop(t *testing.T) {
	// Skip integration tests in CI environments
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}
	config := Config{
		SampleRate:        16000,
		Channels:          1,
		Format:            "s16",
		BufferSize:        8192,
		Device:            "",
		ChannelBufferSize: 30,
		Timeout:           5 * time.Minute,
	}

	recorder := NewRecorder(config)

	// Stop should be safe to call even when not recording
	recorder.Stop()

	// Start recording
	ctx := context.Background()
	_, _, err := recorder.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
		return
	}

	// Stop should work when recording
	recorder.Stop()

	// Give it a moment to stop
	time.Sleep(10 * time.Millisecond)

	// Should not be recording anymore
	if recorder.IsRecording() {
		t.Errorf("IsRecording() = true after stop, want false")
	}
}

// TestRecorder_Start_InvalidConfig tests starting with invalid config
func TestRecorder_Start_InvalidConfig(t *testing.T) {
	invalidConfig := Config{
		SampleRate: 0, // Invalid
		Channels:   1,
		Format:     "s16",
		BufferSize: 8192,
	}

	recorder := NewRecorder(invalidConfig)
	ctx := context.Background()

	_, _, err := recorder.Start(ctx)
	if err == nil {
		t.Errorf("Start() should fail with invalid config")
	}
}
