package daemon

import (
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/pipeline"
)

func TestNew(t *testing.T) {
	// Set up a temporary config directory
	tempDir := t.TempDir()
	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		if originalConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
		}
	}()

	// Create a basic config file
	configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	configContent := `[recording]
sample_rate = 16000
channels = 1
format = "s16"
buffer_size = 8192
channel_buffer_size = 30
timeout = "5m"

[transcription]
provider = "openai"
api_key = "test-key"
model = "whisper-1"

[injection]
mode = "fallback"
wtype_timeout = "5s"
clipboard_timeout = "3s"

[notifications]
enabled = true
type = "log"`
	os.WriteFile(configPath, []byte(configContent), 0644)

	daemon, err := New()
	if err != nil {
		t.Errorf("New() error = %v", err)
		return
	}

	if daemon == nil {
		t.Errorf("New() returned nil")
		return
	}

	// Test that daemon has required components
	if daemon.notifier == nil {
		t.Errorf("Daemon notifier is nil")
	}

	if daemon.configMgr == nil {
		t.Errorf("Daemon config manager is nil")
	}
}

func TestDaemon_Status(t *testing.T) {
	// Set up a temporary config directory
	tempDir := t.TempDir()
	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		if originalConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
		}
	}()

	// Create a basic config file
	configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	configContent := `[recording]
sample_rate = 16000
channels = 1
format = "s16"
buffer_size = 8192
channel_buffer_size = 30
timeout = "5m"

[transcription]
provider = "openai"
api_key = "test-key"
model = "whisper-1"

[injection]
mode = "fallback"
wtype_timeout = "5s"
clipboard_timeout = "3s"

[notifications]
enabled = true
type = "log"`
	os.WriteFile(configPath, []byte(configContent), 0644)

	daemon, err := New()
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test initial status (should be idle with no pipeline)
	status := daemon.status()
	if status != "idle" {
		t.Errorf("Initial status = %s, want idle", status)
	}
}

func TestDaemon_Toggle(t *testing.T) {
	// Set up a temporary config directory
	tempDir := t.TempDir()
	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		if originalConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
		}
	}()

	// Create a basic config file
	configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	configContent := `[recording]
sample_rate = 16000
channels = 1
format = "s16"
buffer_size = 8192
channel_buffer_size = 30
timeout = "5m"

[transcription]
provider = "openai"
api_key = "test-key"
model = "whisper-1"

[injection]
mode = "fallback"
wtype_timeout = "5s"
clipboard_timeout = "3s"

[notifications]
enabled = true
type = "log"`
	os.WriteFile(configPath, []byte(configContent), 0644)

	daemon, err := New()
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test toggle from idle to recording
	daemon.toggle()
	status := daemon.status()
	t.Logf("Status after first toggle = %s", status)

	// Test toggle from recording to idle (abort)
	daemon.toggle()
	status = daemon.status()
	t.Logf("Status after second toggle = %s", status)
}

func TestDaemon_Handle(t *testing.T) {
	// Set up a temporary config directory
	tempDir := t.TempDir()
	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		if originalConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
		}
	}()

	// Create a basic config file
	configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	configContent := `[recording]
sample_rate = 16000
channels = 1
format = "s16"
buffer_size = 8192
channel_buffer_size = 30
timeout = "5m"

[transcription]
provider = "openai"
api_key = "test-key"
model = "whisper-1"

[injection]
mode = "fallback"
wtype_timeout = "5s"
clipboard_timeout = "3s"

[notifications]
enabled = true
type = "log"`
	os.WriteFile(configPath, []byte(configContent), 0644)

	daemon, err := New()
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test status command (simpler test without goroutines)
	t.Run("status_command", func(t *testing.T) {
		mockConn := &MockConn{
			readData:  []byte("s\n"),
			writeData: []byte{},
		}

		// Initialize WaitGroup to avoid panic
		daemon.wg.Add(1)

		// Handle the command
		daemon.handle(mockConn)

		// Check response
		response := string(mockConn.writeData)
		if response != "STATUS status=idle\n" {
			t.Errorf("handle() response = %q, want %q", response, "STATUS status=idle\n")
		}
	})
}

// MockConn implements net.Conn for testing
type MockConn struct {
	readData  []byte
	writeData []byte
	readPos   int
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	if m.readPos >= len(m.readData) {
		return 0, io.EOF
	}
	n = copy(b, m.readData[m.readPos:])
	m.readPos += n
	return n, nil
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	m.writeData = append(m.writeData, b...)
	return len(b), nil
}

func (m *MockConn) Close() error                       { return nil }
func (m *MockConn) LocalAddr() net.Addr                { return nil }
func (m *MockConn) RemoteAddr() net.Addr               { return nil }
func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestDaemon_OnConfigReload(t *testing.T) {
	// Set up a temporary config directory
	tempDir := t.TempDir()
	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		if originalConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
		}
	}()

	// Create a basic config file
	configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	configContent := `[recording]
sample_rate = 16000
channels = 1
format = "s16"
buffer_size = 8192
channel_buffer_size = 30
timeout = "5m"

[transcription]
provider = "openai"
api_key = "test-key"
model = "whisper-1"

[injection]
mode = "fallback"
wtype_timeout = "5s"
clipboard_timeout = "3s"

[notifications]
enabled = true
type = "log"`
	os.WriteFile(configPath, []byte(configContent), 0644)

	daemon, err := New()
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test onConfigReload method
	daemon.onConfigReload()

	// Verify that the method completes without panicking
	// (We can't easily test the internal state changes without more complex mocking)
}

func TestDaemon_StopPipeline(t *testing.T) {
	// Set up a temporary config directory
	tempDir := t.TempDir()
	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		if originalConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
		}
	}()

	// Create a basic config file
	configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	configContent := `[recording]
sample_rate = 16000
channels = 1
format = "s16"
buffer_size = 8192
channel_buffer_size = 30
timeout = "5m"

[transcription]
provider = "openai"
api_key = "test-key"
model = "whisper-1"

[injection]
mode = "fallback"
wtype_timeout = "5s"
clipboard_timeout = "3s"

[notifications]
enabled = true
type = "log"`
	os.WriteFile(configPath, []byte(configContent), 0644)

	daemon, err := New()
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test stopPipeline with nil pipeline
	daemon.stopPipeline()

	// Test stopPipeline with a mock pipeline
	// (This is simplified since we can't easily mock the pipeline interface)
	daemon.mu.Lock()
	daemon.pipeline = &MockPipeline{}
	daemon.mu.Unlock()

	daemon.stopPipeline()

	// Verify pipeline is set to nil
	daemon.mu.RLock()
	if daemon.pipeline != nil {
		t.Errorf("Pipeline should be nil after stopPipeline")
	}
	daemon.mu.RUnlock()
}

func TestDaemon_Handle_Commands(t *testing.T) {
	// Set up a temporary config directory
	tempDir := t.TempDir()
	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		if originalConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
		}
	}()

	// Create a basic config file
	configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	configContent := `[recording]
sample_rate = 16000
channels = 1
format = "s16"
buffer_size = 8192
channel_buffer_size = 30
timeout = "5m"

[transcription]
provider = "openai"
api_key = "test-key"
model = "whisper-1"

[injection]
mode = "fallback"
wtype_timeout = "5s"
clipboard_timeout = "3s"

[notifications]
enabled = true
type = "log"`
	os.WriteFile(configPath, []byte(configContent), 0644)

	daemon, err := New()
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{"toggle_command", "t\n", "OK toggled\n"},
		{"version_command", "v\n", "STATUS proto="},
		{"quit_command", "q\n", "OK quitting\n"},
		{"unknown_command", "x\n", "ERR unknown="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &MockConn{
				readData:  []byte(tt.command),
				writeData: []byte{},
			}

			// Initialize WaitGroup to avoid panic
			daemon.wg.Add(1)

			// Handle the command
			daemon.handle(mockConn)

			// Check response
			response := string(mockConn.writeData)
			if tt.name == "version_command" {
				if len(response) == 0 || !((len(response) >= 12 && response[:12] == "STATUS proto=") || (len(response) >= 13 && response[:13] == "STATUS proto=")) {
					t.Errorf("handle() response = %q, want prefix %q", response, "STATUS proto=")
				}
			} else if tt.name == "unknown_command" {
				if len(response) == 0 || !((len(response) >= 12 && response[:12] == "ERR unknown=") || (len(response) >= 13 && response[:13] == "ERR unknown=")) {
					t.Errorf("handle() response = %q, want prefix %q", response, "ERR unknown=")
				}
			} else if response != tt.expected {
				t.Errorf("handle() response = %q, want %q", response, tt.expected)
			}
		})
	}
}

// MockPipeline implements pipeline.Pipeline for testing
type MockPipeline struct{}

func (m *MockPipeline) Run(ctx context.Context) {}
func (m *MockPipeline) Stop()                   {}
func (m *MockPipeline) Status() pipeline.Status { return pipeline.Idle }
func (m *MockPipeline) GetErrorCh() <-chan pipeline.PipelineError {
	return make(chan pipeline.PipelineError)
}
func (m *MockPipeline) GetActionCh() chan<- pipeline.Action { return make(chan pipeline.Action) }
