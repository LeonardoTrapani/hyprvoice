package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/notify"
)

// createTestConfig returns a valid configuration for testing
func createTestConfig() *Config {
	return &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			Device:            "",
			ChannelBufferSize: 30,
			Timeout:           5 * time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Language: "",
			Model:    "whisper-1",
		},
		Injection: InjectionConfig{
			Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: NotificationsConfig{
			Enabled: true,
			Type:    "log",
		},
	}
}

// createTestConfigWithInvalidValues returns a config with invalid values for testing validation
func createTestConfigWithInvalidValues() *Config {
	return &Config{
		Recording: RecordingConfig{
			SampleRate:        0,  // Invalid
			Channels:          0,  // Invalid
			Format:            "", // Invalid
			BufferSize:        0,  // Invalid
			ChannelBufferSize: 0,  // Invalid
			Timeout:           0,  // Invalid
		},
		Transcription: TranscriptionConfig{
			Provider: "", // Invalid
			APIKey:   "", // Invalid
			Model:    "", // Invalid
		},
		Injection: InjectionConfig{
			Backends: []string{"invalid"}, YdotoolTimeout: 5 * time.Second, // Invalid
			WtypeTimeout:     0, // Invalid
			ClipboardTimeout: 0, // Invalid
		},
		Notifications: NotificationsConfig{
			Type: "invalid", // Invalid
		},
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  createTestConfig(),
			wantErr: false,
		},
		{
			name:    "invalid config",
			config:  createTestConfigWithInvalidValues(),
			wantErr: true,
		},
		{
			name: "invalid recording sample rate",
			config: &Config{
				Recording: RecordingConfig{
					SampleRate:        0,
					Channels:          1,
					Format:            "s16",
					BufferSize:        8192,
					ChannelBufferSize: 30,
					Timeout:           time.Minute,
				},
				Transcription: TranscriptionConfig{
					Provider: "openai",
					APIKey:   "test-key",
					Model:    "whisper-1",
				},
				Injection: InjectionConfig{
					Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
					WtypeTimeout:     time.Second,
					ClipboardTimeout: time.Second,
				},
				Notifications: NotificationsConfig{
					Type: "log",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid transcription provider",
			config: &Config{
				Recording: RecordingConfig{
					SampleRate:        16000,
					Channels:          1,
					Format:            "s16",
					BufferSize:        8192,
					ChannelBufferSize: 30,
					Timeout:           time.Minute,
				},
				Transcription: TranscriptionConfig{
					Provider: "",
					APIKey:   "test-key",
					Model:    "whisper-1",
				},
				Injection: InjectionConfig{
					Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
					WtypeTimeout:     time.Second,
					ClipboardTimeout: time.Second,
				},
				Notifications: NotificationsConfig{
					Type: "log",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid injection mode",
			config: &Config{
				Recording: RecordingConfig{
					SampleRate:        16000,
					Channels:          1,
					Format:            "s16",
					BufferSize:        8192,
					ChannelBufferSize: 30,
					Timeout:           time.Minute,
				},
				Transcription: TranscriptionConfig{
					Provider: "openai",
					APIKey:   "test-key",
					Model:    "whisper-1",
				},
				Injection: InjectionConfig{
					Backends: []string{"invalid"}, YdotoolTimeout: 5 * time.Second,
					WtypeTimeout:     time.Second,
					ClipboardTimeout: time.Second,
				},
				Notifications: NotificationsConfig{
					Type: "log",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid notification type",
			config: &Config{
				Recording: RecordingConfig{
					SampleRate:        16000,
					Channels:          1,
					Format:            "s16",
					BufferSize:        8192,
					ChannelBufferSize: 30,
					Timeout:           time.Minute,
				},
				Transcription: TranscriptionConfig{
					Provider: "openai",
					APIKey:   "test-key",
					Model:    "whisper-1",
				},
				Injection: InjectionConfig{
					Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
					WtypeTimeout:     time.Second,
					ClipboardTimeout: time.Second,
				},
				Notifications: NotificationsConfig{
					Type: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "valid language codes",
			config: &Config{
				Recording: RecordingConfig{
					SampleRate:        16000,
					Channels:          1,
					Format:            "s16",
					BufferSize:        8192,
					ChannelBufferSize: 30,
					Timeout:           time.Minute,
				},
				Transcription: TranscriptionConfig{
					Provider: "openai",
					APIKey:   "test-key",
					Language: "en",
					Model:    "whisper-1",
				},
				Injection: InjectionConfig{
					Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
					WtypeTimeout:     time.Second,
					ClipboardTimeout: time.Second,
				},
				Notifications: NotificationsConfig{
					Type: "log",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid language code",
			config: &Config{
				Recording: RecordingConfig{
					SampleRate:        16000,
					Channels:          1,
					Format:            "s16",
					BufferSize:        8192,
					ChannelBufferSize: 30,
					Timeout:           time.Minute,
				},
				Transcription: TranscriptionConfig{
					Provider: "openai",
					APIKey:   "test-key",
					Language: "invalid",
					Model:    "whisper-1",
				},
				Injection: InjectionConfig{
					Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
					WtypeTimeout:     time.Second,
					ClipboardTimeout: time.Second,
				},
				Notifications: NotificationsConfig{
					Type: "log",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Load(t *testing.T) {
	// Test that Load creates default config when none exists
	t.Run("creates default config when none exists", func(t *testing.T) {
		tempDir := t.TempDir()
		originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
		originalAPIKey := os.Getenv("OPENAI_API_KEY")
		os.Setenv("XDG_CONFIG_HOME", tempDir)
		os.Setenv("OPENAI_API_KEY", "test-api-key") // Set test API key for validation
		defer func() {
			if originalConfigDir == "" {
				os.Unsetenv("XDG_CONFIG_HOME")
			} else {
				os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
			}
			if originalAPIKey == "" {
				os.Unsetenv("OPENAI_API_KEY")
			} else {
				os.Setenv("OPENAI_API_KEY", originalAPIKey)
			}
		}()

		config, err := Load()
		if err != nil {
			t.Errorf("Load() error = %v", err)
			return
		}

		// Verify the loaded config is valid
		if err := config.Validate(); err != nil {
			t.Errorf("Loaded config is invalid: %v", err)
		}

		// Verify config file was created
		configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Load() did not create config file")
		}
	})

	// Test that Load works with existing valid config
	t.Run("loads existing valid config", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")

		// Create directory and config file
		err := os.MkdirAll(filepath.Dir(configPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create config directory: %v", err)
		}

		validConfig := `[recording]
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
backends = ["ydotool", "wtype", "clipboard"]
ydotool_timeout = "5s"
wtype_timeout = "5s"
clipboard_timeout = "3s"

[notifications]
enabled = true
type = "log"`

		err = os.WriteFile(configPath, []byte(validConfig), 0644)
		if err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
		os.Setenv("XDG_CONFIG_HOME", tempDir)
		defer func() {
			if originalConfigDir == "" {
				os.Unsetenv("XDG_CONFIG_HOME")
			} else {
				os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
			}
		}()

		config, err := Load()
		if err != nil {
			t.Errorf("Load() error = %v", err)
			return
		}

		// Verify the loaded config is valid
		if err := config.Validate(); err != nil {
			t.Errorf("Loaded config is invalid: %v", err)
		}

		// Verify specific values were loaded
		if config.Recording.SampleRate != 16000 {
			t.Errorf("Expected SampleRate 16000, got %d", config.Recording.SampleRate)
		}
		if config.Transcription.Provider != "openai" {
			t.Errorf("Expected Provider 'openai', got %s", config.Transcription.Provider)
		}
	})

	// Test migration from legacy mode config
	t.Run("migrates legacy mode=fallback to backends", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")

		err := os.MkdirAll(filepath.Dir(configPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create config directory: %v", err)
		}

		legacyConfig := `[recording]
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

		err = os.WriteFile(configPath, []byte(legacyConfig), 0644)
		if err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
		os.Setenv("XDG_CONFIG_HOME", tempDir)
		defer func() {
			if originalConfigDir == "" {
				os.Unsetenv("XDG_CONFIG_HOME")
			} else {
				os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
			}
		}()

		config, err := Load()
		if err != nil {
			t.Errorf("Load() error = %v", err)
			return
		}

		// Should have migrated to backends
		expectedBackends := []string{"wtype", "clipboard"}
		if len(config.Injection.Backends) != len(expectedBackends) {
			t.Errorf("Expected %d backends, got %d", len(expectedBackends), len(config.Injection.Backends))
		}
		for i, b := range expectedBackends {
			if i < len(config.Injection.Backends) && config.Injection.Backends[i] != b {
				t.Errorf("Expected backend[%d]=%s, got %s", i, b, config.Injection.Backends[i])
			}
		}

		// Should have set default ydotool timeout
		if config.Injection.YdotoolTimeout != 5*time.Second {
			t.Errorf("Expected YdotoolTimeout=5s, got %v", config.Injection.YdotoolTimeout)
		}

		// Verify it passes validation
		if err := config.Validate(); err != nil {
			t.Errorf("Migrated config is invalid: %v", err)
		}
	})

	t.Run("migrates legacy mode=clipboard to backends", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")

		err := os.MkdirAll(filepath.Dir(configPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create config directory: %v", err)
		}

		legacyConfig := `[recording]
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
mode = "clipboard"
wtype_timeout = "5s"
clipboard_timeout = "3s"

[notifications]
enabled = true
type = "log"`

		err = os.WriteFile(configPath, []byte(legacyConfig), 0644)
		if err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
		os.Setenv("XDG_CONFIG_HOME", tempDir)
		defer func() {
			if originalConfigDir == "" {
				os.Unsetenv("XDG_CONFIG_HOME")
			} else {
				os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
			}
		}()

		config, err := Load()
		if err != nil {
			t.Errorf("Load() error = %v", err)
			return
		}

		expectedBackends := []string{"clipboard"}
		if len(config.Injection.Backends) != len(expectedBackends) {
			t.Errorf("Expected %d backends, got %d", len(expectedBackends), len(config.Injection.Backends))
		}

		if err := config.Validate(); err != nil {
			t.Errorf("Migrated config is invalid: %v", err)
		}
	})

	t.Run("migrates legacy mode=type to backends", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")

		err := os.MkdirAll(filepath.Dir(configPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create config directory: %v", err)
		}

		legacyConfig := `[recording]
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
mode = "type"
wtype_timeout = "5s"
clipboard_timeout = "3s"

[notifications]
enabled = true
type = "log"`

		err = os.WriteFile(configPath, []byte(legacyConfig), 0644)
		if err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
		os.Setenv("XDG_CONFIG_HOME", tempDir)
		defer func() {
			if originalConfigDir == "" {
				os.Unsetenv("XDG_CONFIG_HOME")
			} else {
				os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
			}
		}()

		config, err := Load()
		if err != nil {
			t.Errorf("Load() error = %v", err)
			return
		}

		expectedBackends := []string{"wtype"}
		if len(config.Injection.Backends) != len(expectedBackends) {
			t.Errorf("Expected %d backends, got %d", len(expectedBackends), len(config.Injection.Backends))
		}

		if err := config.Validate(); err != nil {
			t.Errorf("Migrated config is invalid: %v", err)
		}
	})
}

func TestConfig_SaveDefaultConfig(t *testing.T) {
	// Override the config path by setting environment variable
	tempDir := t.TempDir()
	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	os.Setenv("OPENAI_API_KEY", "test-api-key") // Set test API key for validation
	defer func() {
		if originalConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
		}
		if originalAPIKey == "" {
			os.Unsetenv("OPENAI_API_KEY")
		} else {
			os.Setenv("OPENAI_API_KEY", originalAPIKey)
		}
	}()

	err := SaveDefaultConfig()
	if err != nil {
		t.Errorf("SaveDefaultConfig() error = %v", err)
		return
	}

	// Verify file was created
	configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("SaveDefaultConfig() did not create config file")
		return
	}

	// Verify file content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Errorf("Failed to read created config file: %v", err)
		return
	}

	if len(content) == 0 {
		t.Errorf("SaveDefaultConfig() created empty config file")
		return
	}

	// Verify it's valid TOML
	config, err := Load()
	if err != nil {
		t.Errorf("SaveDefaultConfig() created invalid config: %v", err)
		return
	}

	// Verify validation passes
	if err := config.Validate(); err != nil {
		t.Errorf("SaveDefaultConfig() created invalid config: %v", err)
	}
}

func TestConfig_ConversionMethods(t *testing.T) {
	config := createTestConfig()

	t.Run("ToRecordingConfig", func(t *testing.T) {
		recordingConfig := config.ToRecordingConfig()

		if recordingConfig.SampleRate != config.Recording.SampleRate {
			t.Errorf("SampleRate mismatch: got %d, want %d", recordingConfig.SampleRate, config.Recording.SampleRate)
		}
		if recordingConfig.Channels != config.Recording.Channels {
			t.Errorf("Channels mismatch: got %d, want %d", recordingConfig.Channels, config.Recording.Channels)
		}
		if recordingConfig.Format != config.Recording.Format {
			t.Errorf("Format mismatch: got %s, want %s", recordingConfig.Format, config.Recording.Format)
		}
	})

	t.Run("ToTranscriberConfig", func(t *testing.T) {
		transcriberConfig := config.ToTranscriberConfig()

		if transcriberConfig.Provider != config.Transcription.Provider {
			t.Errorf("Provider mismatch: got %s, want %s", transcriberConfig.Provider, config.Transcription.Provider)
		}
		if transcriberConfig.APIKey != config.Transcription.APIKey {
			t.Errorf("APIKey mismatch: got %s, want %s", transcriberConfig.APIKey, config.Transcription.APIKey)
		}
		if transcriberConfig.Language != config.Transcription.Language {
			t.Errorf("Language mismatch: got %s, want %s", transcriberConfig.Language, config.Transcription.Language)
		}
		if transcriberConfig.Model != config.Transcription.Model {
			t.Errorf("Model mismatch: got %s, want %s", transcriberConfig.Model, config.Transcription.Model)
		}
	})

	t.Run("ToInjectionConfig", func(t *testing.T) {
		injectionConfig := config.ToInjectionConfig()

		if len(injectionConfig.Backends) != len(config.Injection.Backends) {
			t.Errorf("Backends length mismatch: got %d, want %d", len(injectionConfig.Backends), len(config.Injection.Backends))
		}
		if injectionConfig.YdotoolTimeout != config.Injection.YdotoolTimeout {
			t.Errorf("YdotoolTimeout mismatch: got %v, want %v", injectionConfig.YdotoolTimeout, config.Injection.YdotoolTimeout)
		}
		if injectionConfig.WtypeTimeout != config.Injection.WtypeTimeout {
			t.Errorf("WtypeTimeout mismatch: got %v, want %v", injectionConfig.WtypeTimeout, config.Injection.WtypeTimeout)
		}
		if injectionConfig.ClipboardTimeout != config.Injection.ClipboardTimeout {
			t.Errorf("ClipboardTimeout mismatch: got %v, want %v", injectionConfig.ClipboardTimeout, config.Injection.ClipboardTimeout)
		}
	})
}

func TestValidateModelLanguageCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		model       string
		langCode    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "auto language always passes",
			provider: "groq",
			model:    "distil-whisper-large-v3-en",
			langCode: "",
			wantErr:  false,
		},
		{
			name:     "english model supports english",
			provider: "groq",
			model:    "distil-whisper-large-v3-en",
			langCode: "en",
			wantErr:  false,
		},
		{
			name:        "english model rejects spanish",
			provider:    "groq",
			model:       "distil-whisper-large-v3-en",
			langCode:    "es",
			wantErr:     true,
			errContains: "does not support language 'es'",
		},
		{
			name:     "multilingual model supports spanish",
			provider: "groq",
			model:    "whisper-large-v3",
			langCode: "es",
			wantErr:  false,
		},
		{
			name:        "whisper-cpp english-only rejects french",
			provider:    "whisper-cpp",
			model:       "base.en",
			langCode:    "fr",
			wantErr:     true,
			errContains: "does not support language 'fr'",
		},
		{
			name:     "whisper-cpp multilingual supports french",
			provider: "whisper-cpp",
			model:    "base",
			langCode: "fr",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelLanguageCompatibility(tt.provider, tt.model, tt.langCode)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, should contain %q", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	// Override user config dir for testing using environment variable
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

	path, err := GetConfigPath()
	if err != nil {
		t.Errorf("GetConfigPath() error = %v", err)
		return
	}

	expectedPath := filepath.Join(tempDir, "hyprvoice", "config.toml")
	if path != expectedPath {
		t.Errorf("GetConfigPath() = %s, want %s", path, expectedPath)
	}

	// Verify directory was created
	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		t.Errorf("GetConfigPath() did not create config directory")
	}
}

func TestConfig_ToTranscriberConfig_WithEnvVar(t *testing.T) {
	config := &Config{
		Transcription: TranscriptionConfig{
			Provider: "openai",
			APIKey:   "", // Empty API key to test env var fallback
			Language: "en",
			Model:    "whisper-1",
		},
	}

	// Set environment variable
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "env-api-key")
	defer func() {
		if originalAPIKey == "" {
			os.Unsetenv("OPENAI_API_KEY")
		} else {
			os.Setenv("OPENAI_API_KEY", originalAPIKey)
		}
	}()

	transcriberConfig := config.ToTranscriberConfig()

	if transcriberConfig.APIKey != "env-api-key" {
		t.Errorf("Expected APIKey from env var 'env-api-key', got %s", transcriberConfig.APIKey)
	}
}

func TestConfig_ToTranscriberConfig_WithoutEnvVar(t *testing.T) {
	config := &Config{
		Transcription: TranscriptionConfig{
			Provider: "openai",
			APIKey:   "config-api-key", // Config has API key
			Language: "en",
			Model:    "whisper-1",
		},
	}

	// Ensure environment variable is not set
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	defer func() {
		if originalAPIKey != "" {
			os.Setenv("OPENAI_API_KEY", originalAPIKey)
		}
	}()

	transcriberConfig := config.ToTranscriberConfig()

	if transcriberConfig.APIKey != "config-api-key" {
		t.Errorf("Expected APIKey from config 'config-api-key', got %s", transcriberConfig.APIKey)
	}
}

func TestConfig_Load_InvalidTOML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")

	// Create directory and invalid config file
	err := os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	invalidConfig := `[recording]
sample_rate = "invalid_number"`

	err = os.WriteFile(configPath, []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		if originalConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
		}
	}()

	_, err = Load()
	if err == nil {
		t.Errorf("Load() should have failed with invalid TOML")
	}
}

func TestConfig_Validate_OpenAI_WithoutAPIKey(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "openai",
			APIKey:   "", // No API key
			Model:    "whisper-1",
		},
		Injection: InjectionConfig{
			Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
			WtypeTimeout:     time.Second,
			ClipboardTimeout: time.Second,
		},
		Notifications: NotificationsConfig{
			Type: "log",
		},
	}

	// Ensure environment variable is not set
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	defer func() {
		if originalAPIKey != "" {
			os.Setenv("OPENAI_API_KEY", originalAPIKey)
		}
	}()

	err := config.Validate()
	if err == nil {
		t.Errorf("Validate() should have failed without OpenAI API key")
	}
}

func TestConfig_Validate_OpenAI_WithEnvVarAPIKey(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "openai",
			APIKey:   "", // No API key in config
			Model:    "whisper-1",
		},
		Injection: InjectionConfig{
			Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
			WtypeTimeout:     time.Second,
			ClipboardTimeout: time.Second,
		},
		Notifications: NotificationsConfig{
			Type: "log",
		},
	}

	// Set environment variable
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "env-api-key")
	defer func() {
		if originalAPIKey == "" {
			os.Unsetenv("OPENAI_API_KEY")
		} else {
			os.Setenv("OPENAI_API_KEY", originalAPIKey)
		}
	}()

	err := config.Validate()
	if err != nil {
		t.Errorf("Validate() should have passed with OpenAI API key from environment: %v", err)
	}
}

func TestConfig_Validate_RecordingTimeout(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           0, // Invalid timeout
		},
		Transcription: TranscriptionConfig{
			Provider: "openai",
			APIKey:   "test-key",
			Model:    "whisper-1",
		},
		Injection: InjectionConfig{
			Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
			WtypeTimeout:     time.Second,
			ClipboardTimeout: time.Second,
		},
		Notifications: NotificationsConfig{
			Type: "log",
		},
	}

	err := config.Validate()
	if err == nil {
		t.Errorf("Validate() should have failed with invalid recording timeout")
	}
}

func TestConfig_Validate_InjectionTimeouts(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "openai",
			APIKey:   "test-key",
			Model:    "whisper-1",
		},
		Injection: InjectionConfig{
			Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
			WtypeTimeout:     0, // Invalid timeout
			ClipboardTimeout: 0, // Invalid timeout
		},
		Notifications: NotificationsConfig{
			Type: "log",
		},
	}

	err := config.Validate()
	if err == nil {
		t.Errorf("Validate() should have failed with invalid injection timeouts")
	}
}

func TestConfig_Validate_RecordingBufferSizes(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        0, // Invalid buffer size
			ChannelBufferSize: 0, // Invalid buffer size
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "openai",
			APIKey:   "test-key",
			Model:    "whisper-1",
		},
		Injection: InjectionConfig{
			Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
			WtypeTimeout:     time.Second,
			ClipboardTimeout: time.Second,
		},
		Notifications: NotificationsConfig{
			Type: "log",
		},
	}

	err := config.Validate()
	if err == nil {
		t.Errorf("Validate() should have failed with invalid recording buffer sizes")
	}
}

func TestConfig_Validate_GroqTranscription(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "groq-transcription",
			APIKey:   "gsk-test-key",
			Language: "en",
			Model:    "whisper-large-v3",
		},
		Injection: InjectionConfig{
			Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
			WtypeTimeout:     time.Second,
			ClipboardTimeout: time.Second,
		},
		Notifications: NotificationsConfig{
			Type: "log",
		},
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Validate() should have passed with valid groq-transcription config: %v", err)
	}
}

func TestConfig_Validate_GroqTranslation(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "groq-translation",
			APIKey:   "gsk-test-key",
			Language: "es",
			Model:    "whisper-large-v3", // Translation only supports non-turbo
		},
		Injection: InjectionConfig{
			Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
			WtypeTimeout:     time.Second,
			ClipboardTimeout: time.Second,
		},
		Notifications: NotificationsConfig{
			Type: "log",
		},
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Validate() should have passed with valid groq-translation config: %v", err)
	}
}

func TestConfig_Validate_GroqInvalidModel(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "groq-transcription",
			APIKey:   "gsk-test-key",
			Language: "en",
			Model:    "invalid-model",
		},
		Injection: InjectionConfig{
			Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
			WtypeTimeout:     time.Second,
			ClipboardTimeout: time.Second,
		},
		Notifications: NotificationsConfig{
			Type: "log",
		},
	}

	err := config.Validate()
	if err == nil {
		t.Errorf("Validate() should have failed with invalid Groq model")
	}
}

func TestConfig_Validate_GroqWithoutAPIKey(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "groq-transcription",
			APIKey:   "", // No API key
			Model:    "whisper-large-v3",
		},
		Injection: InjectionConfig{
			Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
			WtypeTimeout:     time.Second,
			ClipboardTimeout: time.Second,
		},
		Notifications: NotificationsConfig{
			Type: "log",
		},
	}

	// Ensure environment variable is not set
	originalAPIKey := os.Getenv("GROQ_API_KEY")
	os.Unsetenv("GROQ_API_KEY")
	defer func() {
		if originalAPIKey != "" {
			os.Setenv("GROQ_API_KEY", originalAPIKey)
		}
	}()

	err := config.Validate()
	if err == nil {
		t.Errorf("Validate() should have failed without Groq API key")
	}
}

func TestConfig_Validate_GroqWithEnvVarAPIKey(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "groq-transcription",
			APIKey:   "", // No API key in config
			Model:    "whisper-large-v3",
		},
		Injection: InjectionConfig{
			Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
			WtypeTimeout:     time.Second,
			ClipboardTimeout: time.Second,
		},
		Notifications: NotificationsConfig{
			Type: "log",
		},
	}

	// Set environment variable
	originalAPIKey := os.Getenv("GROQ_API_KEY")
	os.Setenv("GROQ_API_KEY", "gsk-env-api-key")
	defer func() {
		if originalAPIKey == "" {
			os.Unsetenv("GROQ_API_KEY")
		} else {
			os.Setenv("GROQ_API_KEY", originalAPIKey)
		}
	}()

	err := config.Validate()
	if err != nil {
		t.Errorf("Validate() should have passed with Groq API key from environment: %v", err)
	}
}

func TestConfig_ToTranscriberConfig_GroqWithEnvVar(t *testing.T) {
	config := &Config{
		Transcription: TranscriptionConfig{
			Provider: "groq-transcription",
			APIKey:   "", // Empty API key to test env var fallback
			Language: "en",
			Model:    "whisper-large-v3",
		},
	}

	// Set environment variable
	originalAPIKey := os.Getenv("GROQ_API_KEY")
	os.Setenv("GROQ_API_KEY", "gsk-env-api-key")
	defer func() {
		if originalAPIKey == "" {
			os.Unsetenv("GROQ_API_KEY")
		} else {
			os.Setenv("GROQ_API_KEY", originalAPIKey)
		}
	}()

	transcriberConfig := config.ToTranscriberConfig()

	if transcriberConfig.APIKey != "gsk-env-api-key" {
		t.Errorf("Expected APIKey from env var 'gsk-env-api-key', got %s", transcriberConfig.APIKey)
	}
}

func TestConfig_Validate_GroqTranslation_RejectsTurbo(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "groq-translation",
			APIKey:   "gsk-test-key",
			Language: "es",
			Model:    "whisper-large-v3-turbo", // Turbo not supported for translation
		},
		Injection: InjectionConfig{
			Backends: []string{"ydotool", "wtype", "clipboard"}, YdotoolTimeout: 5 * time.Second,
			WtypeTimeout:     time.Second,
			ClipboardTimeout: time.Second,
		},
		Notifications: NotificationsConfig{
			Type: "log",
		},
	}

	err := config.Validate()
	if err == nil {
		t.Error("Validate() should have rejected whisper-large-v3-turbo for groq-translation")
	}
	if err != nil && err.Error() != "invalid model for groq-translation: whisper-large-v3-turbo (must be whisper-large-v3, turbo version not supported for translation)" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestMessagesConfig_Resolve_Defaults(t *testing.T) {
	cfg := createTestConfig()
	msgs := cfg.Notifications.Messages.Resolve()

	// Check defaults are applied
	if msgs[notify.MsgRecordingStarted].Title != "Hyprvoice" {
		t.Errorf("MsgRecordingStarted title = %q, want %q", msgs[notify.MsgRecordingStarted].Title, "Hyprvoice")
	}
	if msgs[notify.MsgRecordingStarted].Body != "Recording Started" {
		t.Errorf("MsgRecordingStarted body = %q, want %q", msgs[notify.MsgRecordingStarted].Body, "Recording Started")
	}
	if msgs[notify.MsgTranscribing].Body != "Recording Ended... Transcribing" {
		t.Errorf("MsgTranscribing body = %q, want %q", msgs[notify.MsgTranscribing].Body, "Recording Ended... Transcribing")
	}
	if msgs[notify.MsgRecordingAborted].IsError != true {
		t.Errorf("MsgRecordingAborted IsError = %v, want true", msgs[notify.MsgRecordingAborted].IsError)
	}
}

func TestMessagesConfig_Resolve_CustomOverrides(t *testing.T) {
	cfg := createTestConfig()
	cfg.Notifications.Messages = MessagesConfig{
		RecordingStarted: MessageConfig{
			Title: "Custom Title",
			Body:  "Custom Body",
		},
		RecordingAborted: MessageConfig{
			Body: "Custom Abort",
		},
	}

	msgs := cfg.Notifications.Messages.Resolve()

	// Custom values should override defaults
	if msgs[notify.MsgRecordingStarted].Title != "Custom Title" {
		t.Errorf("MsgRecordingStarted title = %q, want %q", msgs[notify.MsgRecordingStarted].Title, "Custom Title")
	}
	if msgs[notify.MsgRecordingStarted].Body != "Custom Body" {
		t.Errorf("MsgRecordingStarted body = %q, want %q", msgs[notify.MsgRecordingStarted].Body, "Custom Body")
	}
	if msgs[notify.MsgRecordingAborted].Body != "Custom Abort" {
		t.Errorf("MsgRecordingAborted body = %q, want %q", msgs[notify.MsgRecordingAborted].Body, "Custom Abort")
	}

	// Non-customized messages should still have defaults
	if msgs[notify.MsgTranscribing].Title != "Hyprvoice" {
		t.Errorf("MsgTranscribing title = %q, want %q", msgs[notify.MsgTranscribing].Title, "Hyprvoice")
	}
}

// Tests for new unified provider structure

func TestConfig_ProvidersMap(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "openai",
			Model:    "whisper-1",
		},
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: "sk-provider-key"},
		},
		Injection: InjectionConfig{
			Backends:         []string{"clipboard"},
			YdotoolTimeout:   5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: NotificationsConfig{Type: "log"},
	}

	// Should resolve API key from providers map
	transcriberConfig := config.ToTranscriberConfig()
	if transcriberConfig.APIKey != "sk-provider-key" {
		t.Errorf("Expected APIKey from providers map, got %s", transcriberConfig.APIKey)
	}

	// Validation should pass
	if err := config.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestConfig_ProvidersMapFallbackToLegacy(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "openai",
			APIKey:   "sk-legacy-key", // Legacy field
			Model:    "whisper-1",
		},
		Providers: map[string]ProviderConfig{}, // Empty providers map
		Injection: InjectionConfig{
			Backends:         []string{"clipboard"},
			YdotoolTimeout:   5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: NotificationsConfig{Type: "log"},
	}

	// Should fall back to legacy transcription.api_key
	transcriberConfig := config.ToTranscriberConfig()
	if transcriberConfig.APIKey != "sk-legacy-key" {
		t.Errorf("Expected APIKey from legacy field, got %s", transcriberConfig.APIKey)
	}
}

func TestConfig_LLMConfig(t *testing.T) {
	config := &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			ChannelBufferSize: 30,
			Timeout:           time.Minute,
		},
		Transcription: TranscriptionConfig{
			Provider: "openai",
			Model:    "whisper-1",
		},
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: "sk-test-key"},
		},
		Keywords: []string{"hyprvoice", "Claude"},
		LLM: LLMConfig{
			Enabled:  true,
			Provider: "openai",
			Model:    "gpt-4o-mini",
			PostProcessing: LLMPostProcessingConfig{
				RemoveStutters:    true,
				AddPunctuation:    true,
				FixGrammar:        false,
				RemoveFillerWords: true,
			},
			CustomPrompt: LLMCustomPromptConfig{
				Enabled: true,
				Prompt:  "Format as code",
			},
		},
		Injection: InjectionConfig{
			Backends:         []string{"clipboard"},
			YdotoolTimeout:   5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: NotificationsConfig{Type: "log"},
	}

	// Validate should pass
	if err := config.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	// IsLLMEnabled should return true
	if !config.IsLLMEnabled() {
		t.Error("IsLLMEnabled() should return true")
	}

	// ToLLMConfig should return correct values
	llmConfig := config.ToLLMConfig()
	if llmConfig.Provider != "openai" {
		t.Errorf("LLM provider = %s, want openai", llmConfig.Provider)
	}
	if llmConfig.Model != "gpt-4o-mini" {
		t.Errorf("LLM model = %s, want gpt-4o-mini", llmConfig.Model)
	}
	if llmConfig.APIKey != "sk-test-key" {
		t.Errorf("LLM APIKey = %s, want sk-test-key", llmConfig.APIKey)
	}
	if !llmConfig.RemoveStutters {
		t.Error("RemoveStutters should be true")
	}
	if llmConfig.FixGrammar {
		t.Error("FixGrammar should be false")
	}
	if llmConfig.CustomPrompt != "Format as code" {
		t.Errorf("CustomPrompt = %s, want 'Format as code'", llmConfig.CustomPrompt)
	}
	if len(llmConfig.Keywords) != 2 {
		t.Errorf("Keywords length = %d, want 2", len(llmConfig.Keywords))
	}
}

func TestConfig_LLMValidation(t *testing.T) {
	baseConfig := func() *Config {
		return &Config{
			Recording: RecordingConfig{
				SampleRate:        16000,
				Channels:          1,
				Format:            "s16",
				BufferSize:        8192,
				ChannelBufferSize: 30,
				Timeout:           time.Minute,
			},
			Transcription: TranscriptionConfig{
				Provider: "openai",
				Model:    "whisper-1",
			},
			Providers: map[string]ProviderConfig{
				"openai": {APIKey: "sk-test-key"},
			},
			Injection: InjectionConfig{
				Backends:         []string{"clipboard"},
				YdotoolTimeout:   5 * time.Second,
				WtypeTimeout:     5 * time.Second,
				ClipboardTimeout: 3 * time.Second,
			},
			Notifications: NotificationsConfig{Type: "log"},
		}
	}

	t.Run("LLM enabled without provider fails", func(t *testing.T) {
		config := baseConfig()
		config.LLM.Enabled = true
		config.LLM.Model = "gpt-4o-mini"
		// No provider set

		err := config.Validate()
		if err == nil {
			t.Error("Validate() should fail when LLM enabled without provider")
		}
	})

	t.Run("LLM enabled without model fails", func(t *testing.T) {
		config := baseConfig()
		config.LLM.Enabled = true
		config.LLM.Provider = "openai"
		// No model set

		err := config.Validate()
		if err == nil {
			t.Error("Validate() should fail when LLM enabled without model")
		}
	})

	t.Run("LLM enabled with invalid provider fails", func(t *testing.T) {
		config := baseConfig()
		config.LLM.Enabled = true
		config.LLM.Provider = "invalid"
		config.LLM.Model = "some-model"

		err := config.Validate()
		if err == nil {
			t.Error("Validate() should fail with invalid LLM provider")
		}
	})

	t.Run("LLM disabled skips validation", func(t *testing.T) {
		config := baseConfig()
		config.LLM.Enabled = false
		config.LLM.Provider = "invalid" // Would fail if validated

		err := config.Validate()
		if err != nil {
			t.Errorf("Validate() should pass when LLM disabled: %v", err)
		}
	})

	t.Run("LLM enabled without API key fails", func(t *testing.T) {
		config := baseConfig()
		config.LLM.Enabled = true
		config.LLM.Provider = "groq"
		config.LLM.Model = "llama-3.3-70b-versatile"
		// No groq API key in providers

		// Clear env var
		orig := os.Getenv("GROQ_API_KEY")
		os.Unsetenv("GROQ_API_KEY")
		defer func() {
			if orig != "" {
				os.Setenv("GROQ_API_KEY", orig)
			}
		}()

		err := config.Validate()
		if err == nil {
			t.Error("Validate() should fail when LLM enabled without API key for provider")
		}
	})
}

func TestConfig_MigrateTranscriptionAPIKey(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")

	err := os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Old-style config with api_key in transcription
	oldConfig := `[recording]
sample_rate = 16000
channels = 1
format = "s16"
buffer_size = 8192
channel_buffer_size = 30
timeout = "5m"

[transcription]
provider = "openai"
api_key = "sk-old-style-key"
model = "whisper-1"

[injection]
backends = ["clipboard"]
ydotool_timeout = "5s"
wtype_timeout = "5s"
clipboard_timeout = "3s"

[notifications]
type = "log"`

	err = os.WriteFile(configPath, []byte(oldConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		if originalConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
		}
	}()

	config, err := Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
		return
	}

	// Should have migrated to providers map
	if config.Providers == nil {
		t.Fatal("Providers map should not be nil after migration")
	}
	if config.Providers["openai"].APIKey != "sk-old-style-key" {
		t.Errorf("Expected migrated API key in providers.openai, got %s", config.Providers["openai"].APIKey)
	}

	// Validation should pass
	if err := config.Validate(); err != nil {
		t.Errorf("Validate() should pass after migration: %v", err)
	}

	// ToTranscriberConfig should resolve correctly
	transcriberConfig := config.ToTranscriberConfig()
	if transcriberConfig.APIKey != "sk-old-style-key" {
		t.Errorf("Expected APIKey 'sk-old-style-key', got %s", transcriberConfig.APIKey)
	}
}

func TestConfig_NewStyleConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")

	err := os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// New-style config with providers map
	newConfig := `keywords = ["Claude", "hyprvoice"]

[recording]
sample_rate = 16000
channels = 1
format = "s16"
buffer_size = 8192
channel_buffer_size = 30
timeout = "5m"

[providers.openai]
api_key = "sk-new-style-key"

[providers.groq]
api_key = "gsk_new-groq-key"

[transcription]
provider = "openai"
model = "whisper-1"

[llm]
enabled = true
provider = "groq"
model = "llama-3.3-70b-versatile"

[llm.post_processing]
remove_stutters = true
add_punctuation = true
fix_grammar = true
remove_filler_words = false

[injection]
backends = ["clipboard"]
ydotool_timeout = "5s"
wtype_timeout = "5s"
clipboard_timeout = "3s"

[notifications]
type = "log"`

	err = os.WriteFile(configPath, []byte(newConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		if originalConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigDir)
		}
	}()

	config, err := Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
		return
	}

	// Providers should be loaded
	if config.Providers["openai"].APIKey != "sk-new-style-key" {
		t.Errorf("Expected openai API key, got %s", config.Providers["openai"].APIKey)
	}
	if config.Providers["groq"].APIKey != "gsk_new-groq-key" {
		t.Errorf("Expected groq API key, got %s", config.Providers["groq"].APIKey)
	}

	// Keywords should be loaded
	if len(config.Keywords) != 2 {
		t.Errorf("Expected 2 keywords, got %d", len(config.Keywords))
	}

	// LLM config should be loaded
	if !config.LLM.Enabled {
		t.Error("LLM should be enabled")
	}
	if config.LLM.Provider != "groq" {
		t.Errorf("LLM provider = %s, want groq", config.LLM.Provider)
	}
	if !config.LLM.PostProcessing.RemoveStutters {
		t.Error("RemoveStutters should be true")
	}
	if config.LLM.PostProcessing.RemoveFillerWords {
		t.Error("RemoveFillerWords should be false")
	}

	// Validation should pass
	if err := config.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	// ToTranscriberConfig should use openai key
	transcriberConfig := config.ToTranscriberConfig()
	if transcriberConfig.APIKey != "sk-new-style-key" {
		t.Errorf("Transcriber APIKey = %s, want sk-new-style-key", transcriberConfig.APIKey)
	}

	// ToLLMConfig should use groq key
	llmConfig := config.ToLLMConfig()
	if llmConfig.APIKey != "gsk_new-groq-key" {
		t.Errorf("LLM APIKey = %s, want gsk_new-groq-key", llmConfig.APIKey)
	}
}

func TestConfig_LLMDefaults(t *testing.T) {
	config := &Config{
		LLM: LLMConfig{
			Enabled:  true,
			Provider: "openai",
			Model:    "gpt-4o-mini",
			// PostProcessing left as zero values
		},
	}

	// Simulate what Load() does
	config.applyLLMDefaults()

	// All post-processing options should default to true
	if !config.LLM.PostProcessing.RemoveStutters {
		t.Error("RemoveStutters should default to true")
	}
	if !config.LLM.PostProcessing.AddPunctuation {
		t.Error("AddPunctuation should default to true")
	}
	if !config.LLM.PostProcessing.FixGrammar {
		t.Error("FixGrammar should default to true")
	}
	if !config.LLM.PostProcessing.RemoveFillerWords {
		t.Error("RemoveFillerWords should default to true")
	}
}

func TestConfig_LLMDefaultsPreserveExplicit(t *testing.T) {
	config := &Config{
		LLM: LLMConfig{
			Enabled:  true,
			Provider: "openai",
			Model:    "gpt-4o-mini",
			PostProcessing: LLMPostProcessingConfig{
				RemoveStutters: true, // One is set
				// Others are false
			},
		},
	}

	// Simulate what Load() does
	config.applyLLMDefaults()

	// Should preserve the explicit setting and not override
	if !config.LLM.PostProcessing.RemoveStutters {
		t.Error("RemoveStutters should remain true")
	}
	// Since at least one is true, defaults should NOT be applied
	if config.LLM.PostProcessing.AddPunctuation {
		t.Error("AddPunctuation should remain false (explicit)")
	}
}

func TestConfig_Validate_WhisperCpp(t *testing.T) {
	baseConfig := func() *Config {
		return &Config{
			Recording: RecordingConfig{
				SampleRate:        16000,
				Channels:          1,
				Format:            "s16",
				BufferSize:        8192,
				ChannelBufferSize: 30,
				Timeout:           time.Minute,
			},
			Injection: InjectionConfig{
				Backends:         []string{"clipboard"},
				YdotoolTimeout:   5 * time.Second,
				WtypeTimeout:     5 * time.Second,
				ClipboardTimeout: 3 * time.Second,
			},
			Notifications: NotificationsConfig{Type: "log"},
		}
	}

	t.Run("whisper-cpp valid without API key", func(t *testing.T) {
		config := baseConfig()
		config.Transcription = TranscriptionConfig{
			Provider: "whisper-cpp",
			Model:    "base.en",
			// No API key required
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("Validate() should pass for whisper-cpp without API key: %v", err)
		}
	})

	t.Run("whisper-cpp valid multilingual model", func(t *testing.T) {
		config := baseConfig()
		config.Transcription = TranscriptionConfig{
			Provider: "whisper-cpp",
			Model:    "large-v3",
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("Validate() should pass for whisper-cpp with large-v3: %v", err)
		}
	})

	t.Run("whisper-cpp invalid model", func(t *testing.T) {
		config := baseConfig()
		config.Transcription = TranscriptionConfig{
			Provider: "whisper-cpp",
			Model:    "invalid-model",
		}

		err := config.Validate()
		if err == nil {
			t.Error("Validate() should fail for whisper-cpp with invalid model")
		}
	})

	t.Run("whisper-cpp invalid language", func(t *testing.T) {
		config := baseConfig()
		config.Transcription = TranscriptionConfig{
			Provider: "whisper-cpp",
			Model:    "base.en",
			Language: "invalid-lang",
		}

		err := config.Validate()
		if err == nil {
			t.Error("Validate() should fail for whisper-cpp with invalid language")
		}
	})

	t.Run("whisper-cpp valid with language", func(t *testing.T) {
		config := baseConfig()
		config.Transcription = TranscriptionConfig{
			Provider: "whisper-cpp",
			Model:    "base",
			Language: "en",
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("Validate() should pass for whisper-cpp with valid language: %v", err)
		}
	})
}

func TestConfig_ThreadsDefault(t *testing.T) {
	t.Run("threads defaults to NumCPU-1", func(t *testing.T) {
		config := &Config{
			Transcription: TranscriptionConfig{
				Provider: "whisper-cpp",
				Model:    "base.en",
				Threads:  0, // Not set
			},
		}

		config.applyThreadsDefault()

		expectedThreads := runtime.NumCPU() - 1
		if expectedThreads < 1 {
			expectedThreads = 1
		}

		if config.Transcription.Threads != expectedThreads {
			t.Errorf("Threads = %d, want %d", config.Transcription.Threads, expectedThreads)
		}
	})

	t.Run("explicit threads preserved", func(t *testing.T) {
		config := &Config{
			Transcription: TranscriptionConfig{
				Provider: "whisper-cpp",
				Model:    "base.en",
				Threads:  2, // Explicitly set
			},
		}

		config.applyThreadsDefault()

		if config.Transcription.Threads != 2 {
			t.Errorf("Threads = %d, want 2", config.Transcription.Threads)
		}
	})

	t.Run("threads minimum is 1", func(t *testing.T) {
		config := &Config{
			Transcription: TranscriptionConfig{
				Provider: "whisper-cpp",
				Model:    "base.en",
				Threads:  0,
			},
		}

		config.applyThreadsDefault()

		if config.Transcription.Threads < 1 {
			t.Errorf("Threads = %d, should be at least 1", config.Transcription.Threads)
		}
	})
}

func TestConfig_ToTranscriberConfig_Threads(t *testing.T) {
	config := &Config{
		Transcription: TranscriptionConfig{
			Provider: "whisper-cpp",
			Model:    "base.en",
			Threads:  4,
		},
	}

	transcriberConfig := config.ToTranscriberConfig()

	if transcriberConfig.Threads != 4 {
		t.Errorf("Threads = %d, want 4", transcriberConfig.Threads)
	}
}
