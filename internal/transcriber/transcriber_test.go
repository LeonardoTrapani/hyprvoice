package transcriber

import (
	"context"
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/recording"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	t.Run("default values", func(t *testing.T) {
		if config.Provider != "openai" {
			t.Errorf("default provider should be openai, got %s", config.Provider)
		}
		if config.Language != "it" {
			t.Errorf("default language should be it, got %s", config.Language)
		}
		if config.ChunkSize != 16384 {
			t.Errorf("default chunk size should be 16384, got %d", config.ChunkSize)
		}
		if config.BufferTime != 2*time.Second {
			t.Errorf("default buffer time should be 2s, got %v", config.BufferTime)
		}
		if config.Model != "whisper-1" {
			t.Errorf("default model should be whisper-1, got %s", config.Model)
		}
		if config.APIKey != "" {
			t.Errorf("default API key should be empty, got %s", config.APIKey)
		}
	})
}

func TestNewTranscriber(t *testing.T) {
	t.Run("openai provider with API key", func(t *testing.T) {
		config := Config{
			Provider: "openai",
			APIKey:   "test-api-key",
		}

		transcriber, err := NewTranscriber(config)
		if err != nil {
			t.Fatalf("NewTranscriber failed: %v", err)
		}

		if transcriber == nil {
			t.Fatal("transcriber should not be nil")
		}
	})

	t.Run("openai provider without API key", func(t *testing.T) {
		config := Config{
			Provider: "openai",
			APIKey:   "",
		}

		_, err := NewTranscriber(config)
		if err == nil {
			t.Error("NewTranscriber should fail without API key")
		}

		expectedMsg := "OpenAI API key required"
		if err.Error() != expectedMsg {
			t.Errorf("error message should be %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("unsupported provider", func(t *testing.T) {
		config := Config{
			Provider: "unsupported",
			APIKey:   "test-key",
		}

		_, err := NewTranscriber(config)
		if err == nil {
			t.Error("NewTranscriber should fail with unsupported provider")
		}

		expectedMsg := "unsupported provider: unsupported"
		if err.Error() != expectedMsg {
			t.Errorf("error message should be %q, got %q", expectedMsg, err.Error())
		}
	})
}

func TestOpenAITranscriberCreation(t *testing.T) {
	config := Config{
		Provider:   "openai",
		APIKey:     "test-api-key",
		Language:   "en",
		ChunkSize:  8192,
		BufferTime: 1 * time.Second,
		Model:      "whisper-1",
	}

	transcriber := NewOpenAITranscriber(config)

	t.Run("creation", func(t *testing.T) {
		if transcriber == nil {
			t.Fatal("OpenAI transcriber should not be nil")
		}

		if transcriber.config.APIKey != config.APIKey {
			t.Error("transcriber should store the provided config")
		}

		if transcriber.client == nil {
			t.Error("transcriber should have OpenAI client")
		}

		if transcriber.buffer == nil {
			t.Error("transcriber should have audio buffer")
		}
	})

	t.Run("initial state", func(t *testing.T) {
		if transcriber.transcribing {
			t.Error("transcriber should not be transcribing initially")
		}

		text, err := transcriber.GetTranscription()
		if err != nil {
			t.Errorf("GetTranscription should not error initially: %v", err)
		}

		if text != "" {
			t.Errorf("initial transcription should be empty, got %q", text)
		}
	})
}

func TestAudioBuffer(t *testing.T) {
	config := Config{
		ChunkSize: 1024,
	}
	buffer := &audioBuffer{
		data:    make([]byte, 0, config.ChunkSize*2),
		maxSize: config.ChunkSize,
	}

	t.Run("initial state", func(t *testing.T) {
		if buffer.hasData() {
			t.Error("buffer should not have data initially")
		}

		if buffer.shouldFlush(time.Second) {
			t.Error("buffer should not need flushing initially")
		}

		data := buffer.flush()
		if len(data) != 0 {
			t.Error("flushing empty buffer should return empty data")
		}
	})

	t.Run("add frame", func(t *testing.T) {
		frame := recording.AudioFrame{
			Data:      []byte("test audio data"),
			Timestamp: time.Now(),
		}

		buffer.addFrame(frame)

		if !buffer.hasData() {
			t.Error("buffer should have data after adding frame")
		}

		if len(buffer.data) != len(frame.Data) {
			t.Errorf("buffer data length should be %d, got %d", len(frame.Data), len(buffer.data))
		}

		// Check that data was copied correctly
		for i, b := range buffer.data {
			if b != frame.Data[i] {
				t.Errorf("buffer data[%d] should be %d, got %d", i, frame.Data[i], b)
			}
		}
	})

	t.Run("flush buffer", func(t *testing.T) {
		data := buffer.flush()

		if len(data) == 0 {
			t.Error("flush should return data")
		}

		if buffer.hasData() {
			t.Error("buffer should be empty after flush")
		}

		// Flush again should return empty
		data2 := buffer.flush()
		if len(data2) != 0 {
			t.Error("second flush should return empty data")
		}
	})

	t.Run("buffer overflow", func(t *testing.T) {
		// Fill buffer beyond maxSize
		largeData := make([]byte, config.ChunkSize*3)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		frame := recording.AudioFrame{
			Data:      largeData,
			Timestamp: time.Now(),
		}

		buffer.addFrame(frame)

		// Buffer should be trimmed to maxSize
		if len(buffer.data) > config.ChunkSize {
			t.Errorf("buffer should be trimmed to maxSize %d, got %d", config.ChunkSize, len(buffer.data))
		}
	})

	t.Run("should flush conditions", func(t *testing.T) {
		// Reset buffer for this test
		buffer = &audioBuffer{
			data:    make([]byte, 0, config.ChunkSize*2),
			maxSize: config.ChunkSize,
		}

		// Empty buffer should not flush
		if buffer.shouldFlush(time.Second) {
			t.Error("empty buffer should not need flushing")
		}

		// Add data
		frame := recording.AudioFrame{
			Data:      make([]byte, 10),
			Timestamp: time.Now(),
		}
		buffer.addFrame(frame)

		// Fresh data should not flush immediately
		if buffer.shouldFlush(time.Second) {
			t.Error("fresh data should not need flushing immediately")
		}

		// Old data should flush
		buffer.lastAdd = time.Now().Add(-2 * time.Second)
		if !buffer.shouldFlush(time.Second) {
			t.Error("old data should need flushing")
		}

		// Full buffer should flush regardless of time
		buffer.data = make([]byte, buffer.maxSize)
		buffer.lastAdd = time.Now()
		if !buffer.shouldFlush(time.Hour) {
			t.Error("full buffer should need flushing")
		}
	})
}

func TestOpenAITranscriberLifecycle(t *testing.T) {
	config := Config{
		Provider:   "openai",
		APIKey:     "test-api-key",
		Language:   "en",
		ChunkSize:  1024,
		BufferTime: 100 * time.Millisecond,
		Model:      "whisper-1",
	}

	transcriber := NewOpenAITranscriber(config)

	t.Run("stop before start", func(t *testing.T) {
		err := transcriber.Stop()
		if err != nil {
			t.Errorf("Stop should not error when not started: %v", err)
		}
	})

	t.Run("start transcriber", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		frameCh := make(chan recording.AudioFrame, 10)

		errCh, err := transcriber.Start(ctx, frameCh)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		if errCh == nil {
			t.Fatal("error channel should not be nil")
		}

		// Should be transcribing now
		if !transcriber.transcribing {
			t.Error("transcriber should be marked as transcribing")
		}

		// Stop transcriber
		err = transcriber.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}

		// Should not be transcribing anymore
		if transcriber.transcribing {
			t.Error("transcriber should not be marked as transcribing after stop")
		}
	})

	t.Run("double start", func(t *testing.T) {
		ctx := context.Background()
		frameCh := make(chan recording.AudioFrame, 10)

		// Mark as transcribing manually
		transcriber.transcribing = true
		defer func() { transcriber.transcribing = false }()

		_, err := transcriber.Start(ctx, frameCh)
		if err == nil {
			t.Error("Start should fail when already transcribing")
		}

		expectedMsg := "transcriber: already transcribing"
		if err.Error() != expectedMsg {
			t.Errorf("error message should be %q, got %q", expectedMsg, err.Error())
		}
	})
}

func TestConvertToWAV(t *testing.T) {
	config := Config{
		Provider: "openai",
		APIKey:   "test-api-key",
	}
	transcriber := NewOpenAITranscriber(config)

	t.Run("convert empty audio", func(t *testing.T) {
		rawAudio := []byte{}
		wavData, err := transcriber.convertToWAV(rawAudio)
		if err != nil {
			t.Fatalf("convertToWAV failed: %v", err)
		}

		// WAV header is 44 bytes minimum
		if len(wavData) < 44 {
			t.Errorf("WAV data should be at least 44 bytes, got %d", len(wavData))
		}

		// Check WAV header magic
		if string(wavData[0:4]) != "RIFF" {
			t.Error("WAV should start with RIFF")
		}
		if string(wavData[8:12]) != "WAVE" {
			t.Error("WAV should contain WAVE identifier")
		}
	})

	t.Run("convert audio data", func(t *testing.T) {
		rawAudio := make([]byte, 1024) // 1KB of audio data
		for i := range rawAudio {
			rawAudio[i] = byte(i % 256)
		}

		wavData, err := transcriber.convertToWAV(rawAudio)
		if err != nil {
			t.Fatalf("convertToWAV failed: %v", err)
		}

		expectedSize := 44 + len(rawAudio) // Header + data
		if len(wavData) != expectedSize {
			t.Errorf("WAV data should be %d bytes, got %d", expectedSize, len(wavData))
		}

		// Check that audio data is at the end
		audioDataStart := len(wavData) - len(rawAudio)
		for i, b := range rawAudio {
			if wavData[audioDataStart+i] != b {
				t.Errorf("audio data[%d] should be %d, got %d", i, b, wavData[audioDataStart+i])
			}
		}
	})

	t.Run("WAV header validation", func(t *testing.T) {
		rawAudio := make([]byte, 16) // Small audio sample
		wavData, err := transcriber.convertToWAV(rawAudio)
		if err != nil {
			t.Fatalf("convertToWAV failed: %v", err)
		}

		// Validate WAV header fields
		tests := []struct {
			offset   int
			expected string
			name     string
		}{
			{0, "RIFF", "RIFF identifier"},
			{8, "WAVE", "WAVE identifier"},
			{12, "fmt ", "format chunk identifier"},
			{36, "data", "data chunk identifier"},
		}

		for _, tt := range tests {
			if tt.offset+4 > len(wavData) {
				t.Errorf("WAV data too short for %s", tt.name)
				continue
			}
			actual := string(wavData[tt.offset : tt.offset+4])
			if actual != tt.expected {
				t.Errorf("%s should be %q, got %q", tt.name, tt.expected, actual)
			}
		}
	})
}

func TestTranscriberConcurrency(t *testing.T) {
	config := Config{
		Provider: "openai",
		APIKey:   "test-api-key",
	}
	transcriber := NewOpenAITranscriber(config)

	t.Run("concurrent GetTranscription calls", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					_, _ = transcriber.GetTranscription()
				}
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			select {
			case <-done:
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for concurrent GetTranscription calls")
			}
		}
	})

	t.Run("concurrent Stop calls", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				transcriber.Stop()
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

func TestTranscriptionResult(t *testing.T) {
	t.Run("transcription result creation", func(t *testing.T) {
		text := "Hello, world!"
		timestamp := time.Now()
		isFinal := true

		result := TranscriptionResult{
			Text:      text,
			Timestamp: timestamp,
			IsFinal:   isFinal,
		}

		if result.Text != text {
			t.Errorf("text should be %q, got %q", text, result.Text)
		}
		if !result.Timestamp.Equal(timestamp) {
			t.Errorf("timestamp should be %v, got %v", timestamp, result.Timestamp)
		}
		if result.IsFinal != isFinal {
			t.Errorf("IsFinal should be %v, got %v", isFinal, result.IsFinal)
		}
	})

	t.Run("empty transcription result", func(t *testing.T) {
		result := TranscriptionResult{}

		if result.Text != "" {
			t.Error("empty result text should be empty string")
		}
		if !result.Timestamp.IsZero() {
			t.Error("empty result timestamp should be zero")
		}
		if result.IsFinal {
			t.Error("empty result should not be final")
		}
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("various provider configurations", func(t *testing.T) {
		tests := []struct {
			name        string
			config      Config
			expectError bool
			errorMsg    string
		}{
			{
				name: "valid openai config",
				config: Config{
					Provider: "openai",
					APIKey:   "sk-test-key",
					Language: "en",
					Model:    "whisper-1",
				},
				expectError: false,
			},
			{
				name: "openai without api key",
				config: Config{
					Provider: "openai",
					APIKey:   "",
				},
				expectError: true,
				errorMsg:    "OpenAI API key required",
			},
			{
				name: "unknown provider",
				config: Config{
					Provider: "unknown",
					APIKey:   "test-key",
				},
				expectError: true,
				errorMsg:    "unsupported provider: unknown",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := NewTranscriber(tt.config)

				if tt.expectError {
					if err == nil {
						t.Errorf("expected error for config %+v", tt.config)
					} else if err.Error() != tt.errorMsg {
						t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
					}
				} else {
					if err != nil {
						t.Errorf("unexpected error for config %+v: %v", tt.config, err)
					}
				}
			})
		}
	})
}
