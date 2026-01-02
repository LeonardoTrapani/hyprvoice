package config

import (
	"os"
	"path/filepath"
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

func TestIsValidLanguageCode(t *testing.T) {
	validCodes := []string{"en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko", "zh", "ar", "hi"}
	invalidCodes := []string{"", "invalid", "xx", "123", "EN", "en-us"}

	for _, code := range validCodes {
		t.Run("valid_"+code, func(t *testing.T) {
			if !isValidLanguageCode(code) {
				t.Errorf("isValidLanguageCode(%s) = false, want true", code)
			}
		})
	}

	for _, code := range invalidCodes {
		t.Run("invalid_"+code, func(t *testing.T) {
			if isValidLanguageCode(code) {
				t.Errorf("isValidLanguageCode(%s) = true, want false", code)
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
