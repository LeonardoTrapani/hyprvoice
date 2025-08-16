package recording

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	t.Run("default values", func(t *testing.T) {
		if config.SampleRate != 16000 {
			t.Errorf("default sample rate should be 16000, got %d", config.SampleRate)
		}
		if config.Channels != 1 {
			t.Errorf("default channels should be 1, got %d", config.Channels)
		}
		if config.Format != "s16" {
			t.Errorf("default format should be s16, got %s", config.Format)
		}
		if config.BufferSize != 8192 {
			t.Errorf("default buffer size should be 8192, got %d", config.BufferSize)
		}
		if config.Device != "" {
			t.Errorf("default device should be empty, got %s", config.Device)
		}
		if config.ChannelBufferSize != 30 {
			t.Errorf("default channel buffer size should be 30, got %d", config.ChannelBufferSize)
		}
	})
}

func TestNewRecorder(t *testing.T) {
	config := DefaultConfig()
	recorder := NewRecorder(config)

	t.Run("initial state", func(t *testing.T) {
		if recorder == nil {
			t.Fatal("recorder should not be nil")
		}
		if recorder.IsRecording() {
			t.Error("recorder should not be recording initially")
		}
		if recorder.config.SampleRate != config.SampleRate {
			t.Error("recorder should store the provided config")
		}
	})
}

func TestNewDefaultRecorder(t *testing.T) {
	recorder := NewDefaultRecorder()

	t.Run("default recorder", func(t *testing.T) {
		if recorder == nil {
			t.Fatal("default recorder should not be nil")
		}
		if recorder.IsRecording() {
			t.Error("default recorder should not be recording initially")
		}
		if recorder.config.SampleRate != 16000 {
			t.Error("default recorder should use default config")
		}
	})
}

func TestRecorderValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name:        "valid default config",
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "invalid sample rate",
			config: Config{
				SampleRate:        0,
				Channels:          1,
				Format:            "s16",
				BufferSize:        8192,
				ChannelBufferSize: 30,
			},
			expectError: true,
		},
		{
			name: "negative sample rate",
			config: Config{
				SampleRate:        -1,
				Channels:          1,
				Format:            "s16",
				BufferSize:        8192,
				ChannelBufferSize: 30,
			},
			expectError: true,
		},
		{
			name: "invalid channels",
			config: Config{
				SampleRate:        16000,
				Channels:          0,
				Format:            "s16",
				BufferSize:        8192,
				ChannelBufferSize: 30,
			},
			expectError: true,
		},
		{
			name: "invalid buffer size",
			config: Config{
				SampleRate:        16000,
				Channels:          1,
				Format:            "s16",
				BufferSize:        0,
				ChannelBufferSize: 30,
			},
			expectError: true,
		},
		{
			name: "invalid channel buffer size",
			config: Config{
				SampleRate:        16000,
				Channels:          1,
				Format:            "s16",
				BufferSize:        8192,
				ChannelBufferSize: 0,
			},
			expectError: true,
		},
		{
			name: "empty format",
			config: Config{
				SampleRate:        16000,
				Channels:          1,
				Format:            "",
				BufferSize:        8192,
				ChannelBufferSize: 30,
			},
			expectError: true,
		},
		{
			name: "unaligned buffer size",
			config: Config{
				SampleRate:        16000,
				Channels:          1,
				Format:            "s16",
				BufferSize:        8193, // Not aligned to frame size (2 bytes per sample)
				ChannelBufferSize: 30,
			},
			expectError: false, // Should log warning but not error
		},
		{
			name: "stereo valid config",
			config: Config{
				SampleRate:        48000,
				Channels:          2,
				Format:            "s16",
				BufferSize:        8192, // 4 bytes per frame with 2 channels
				ChannelBufferSize: 30,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := NewRecorder(tt.config)
			err := recorder.validateConfig()

			if tt.expectError && err == nil {
				t.Errorf("expected error for config %+v", tt.config)
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error for config %+v: %v", tt.config, err)
			}
		})
	}
}

func TestRecorderBuildPwRecordArgs(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected []string
	}{
		{
			name:   "default config",
			config: DefaultConfig(),
			expected: []string{
				"--format", "s16",
				"--rate", "16000",
				"--channels", "1",
				"-",
			},
		},
		{
			name: "with device",
			config: Config{
				SampleRate:        48000,
				Channels:          2,
				Format:            "f32",
				Device:            "alsa_output.pci-0000_00_1f.3.analog-stereo",
				BufferSize:        4096,
				ChannelBufferSize: 30,
			},
			expected: []string{
				"--format", "f32",
				"--rate", "48000",
				"--channels", "2",
				"-",
				"--target", "alsa_output.pci-0000_00_1f.3.analog-stereo",
			},
		},
		{
			name: "different sample rate",
			config: Config{
				SampleRate:        44100,
				Channels:          1,
				Format:            "s24",
				BufferSize:        8192,
				ChannelBufferSize: 30,
			},
			expected: []string{
				"--format", "s24",
				"--rate", "44100",
				"--channels", "1",
				"-",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := NewRecorder(tt.config)
			args := recorder.buildPwRecordArgs()

			if len(args) != len(tt.expected) {
				t.Errorf("args length mismatch: got %d, expected %d", len(args), len(tt.expected))
				t.Errorf("got: %v", args)
				t.Errorf("expected: %v", tt.expected)
				return
			}

			for i, arg := range args {
				if arg != tt.expected[i] {
					t.Errorf("arg[%d] mismatch: got %q, expected %q", i, arg, tt.expected[i])
				}
			}
		})
	}
}

func TestRecorderLifecycle(t *testing.T) {
	recorder := NewDefaultRecorder()

	t.Run("initial state", func(t *testing.T) {
		if recorder.IsRecording() {
			t.Error("recorder should not be recording initially")
		}
	})

	t.Run("stop before start", func(t *testing.T) {
		err := recorder.Stop()
		if err != nil {
			t.Errorf("stop should not error when not recording: %v", err)
		}
	})

	// Note: We can't easily test actual recording without PipeWire
	// But we can test the state management
}

func TestAudioFrame(t *testing.T) {
	t.Run("audio frame creation", func(t *testing.T) {
		data := []byte("test audio data")
		timestamp := time.Now()

		frame := AudioFrame{
			Data:      data,
			Timestamp: timestamp,
		}

		if len(frame.Data) != len(data) {
			t.Errorf("frame data length mismatch: got %d, expected %d", len(frame.Data), len(data))
		}

		for i, b := range frame.Data {
			if b != data[i] {
				t.Errorf("frame data[%d] mismatch: got %d, expected %d", i, b, data[i])
			}
		}

		if !frame.Timestamp.Equal(timestamp) {
			t.Errorf("frame timestamp mismatch: got %v, expected %v", frame.Timestamp, timestamp)
		}
	})

	t.Run("empty audio frame", func(t *testing.T) {
		frame := AudioFrame{}

		if frame.Data != nil {
			t.Error("empty frame data should be nil")
		}

		if !frame.Timestamp.IsZero() {
			t.Error("empty frame timestamp should be zero")
		}
	})

	t.Run("large audio frame", func(t *testing.T) {
		data := make([]byte, 65536) // 64KB
		for i := range data {
			data[i] = byte(i % 256)
		}

		frame := AudioFrame{
			Data:      data,
			Timestamp: time.Now(),
		}

		if len(frame.Data) != len(data) {
			t.Errorf("large frame data length mismatch: got %d, expected %d", len(frame.Data), len(data))
		}
	})
}

func TestCheckPipeWireAvailable(t *testing.T) {
	ctx := context.Background()

	t.Run("check pipewire availability", func(t *testing.T) {
		// This test depends on the system having PipeWire installed
		// We'll just verify it doesn't panic and returns some result
		err := CheckPipeWireAvailable(ctx)

		// We can't assert on the specific result since it depends on the system
		// But we can ensure it doesn't panic and returns within reasonable time
		_ = err
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := CheckPipeWireAvailable(ctx)
		// Should return an error due to context cancellation
		if err == nil {
			t.Log("CheckPipeWireAvailable with cancelled context returned nil (may be OK if commands complete quickly)")
		}
	})

	t.Run("timeout context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		err := CheckPipeWireAvailable(ctx)
		// Should return an error due to timeout
		if err == nil {
			t.Log("CheckPipeWireAvailable with timeout context returned nil (may be OK if commands complete quickly)")
		}
	})
}

func TestRecorderConcurrency(t *testing.T) {
	recorder := NewDefaultRecorder()

	t.Run("concurrent IsRecording calls", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					recorder.IsRecording()
				}
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			select {
			case <-done:
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for concurrent IsRecording calls")
			}
		}
	})

	t.Run("concurrent Stop calls", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				recorder.Stop()
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			select {
			case <-done:
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for concurrent Stop calls")
			}
		}
	})
}

func TestConfigEquality(t *testing.T) {
	config1 := DefaultConfig()
	config2 := DefaultConfig()

	t.Run("default configs should be equal", func(t *testing.T) {
		if config1.SampleRate != config2.SampleRate {
			t.Error("sample rates should be equal")
		}
		if config1.Channels != config2.Channels {
			t.Error("channels should be equal")
		}
		if config1.Format != config2.Format {
			t.Error("formats should be equal")
		}
		if config1.BufferSize != config2.BufferSize {
			t.Error("buffer sizes should be equal")
		}
		if config1.Device != config2.Device {
			t.Error("devices should be equal")
		}
		if config1.ChannelBufferSize != config2.ChannelBufferSize {
			t.Error("channel buffer sizes should be equal")
		}
	})

	t.Run("modified config should not be equal", func(t *testing.T) {
		config2.SampleRate = 48000

		if config1.SampleRate == config2.SampleRate {
			t.Error("sample rates should not be equal after modification")
		}
	})
}

func TestRecorderErrorConditions(t *testing.T) {
	t.Run("double start without stop", func(t *testing.T) {
		recorder := NewDefaultRecorder()

		// Mark as recording manually to simulate the condition
		recorder.recording.Store(true)
		defer recorder.recording.Store(false)

		ctx := context.Background()
		_, _, err := recorder.Start(ctx)
		if err == nil {
			t.Error("Start should return error when already recording")
		}

		expectedMsg := "already recording"
		if err.Error() != expectedMsg {
			t.Errorf("error message should be %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("invalid config start", func(t *testing.T) {
		invalidConfig := Config{
			SampleRate: -1, // Invalid
		}
		recorder := NewRecorder(invalidConfig)

		ctx := context.Background()
		_, _, err := recorder.Start(ctx)
		if err == nil {
			t.Error("Start should return error with invalid config")
		}
	})
}
