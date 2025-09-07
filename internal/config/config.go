package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/leonardotrapani/hyprvoice/internal/injection"
	"github.com/leonardotrapani/hyprvoice/internal/recording"
	"github.com/leonardotrapani/hyprvoice/internal/transcriber"
)

type Config struct {
	Recording     RecordingConfig     `toml:"recording"`
	Transcription TranscriptionConfig `toml:"transcription"`
	Injection     InjectionConfig     `toml:"injection"`
	Notifications NotificationsConfig `toml:"notifications"`
}

type RecordingConfig struct {
	SampleRate        int           `toml:"sample_rate"`
	Channels          int           `toml:"channels"`
	Format            string        `toml:"format"`
	BufferSize        int           `toml:"buffer_size"`
	Device            string        `toml:"device"`
	ChannelBufferSize int           `toml:"channel_buffer_size"`
	Timeout           time.Duration `toml:"timeout"`
}

type TranscriptionConfig struct {
	Provider string `toml:"provider"`
	APIKey   string `toml:"api_key"`
	Language string `toml:"language"`
	Model    string `toml:"model"`
}

type InjectionConfig struct {
	Mode             string        `toml:"mode"`
	RestoreClipboard bool          `toml:"restore_clipboard"`
	WtypeTimeout     time.Duration `toml:"wtype_timeout"`
	ClipboardTimeout time.Duration `toml:"clipboard_timeout"`
}

type NotificationsConfig struct {
	Enabled bool   `toml:"enabled"`
	Type    string `toml:"type"` // "desktop", "log", "none"
}

func (c *Config) ToRecordingConfig() recording.Config {
	return recording.Config{
		SampleRate:        c.Recording.SampleRate,
		Channels:          c.Recording.Channels,
		Format:            c.Recording.Format,
		BufferSize:        c.Recording.BufferSize,
		Device:            c.Recording.Device,
		ChannelBufferSize: c.Recording.ChannelBufferSize,
		Timeout:           c.Recording.Timeout,
	}
}

func (c *Config) ToTranscriberConfig() transcriber.Config {
	config := transcriber.Config{
		Provider: c.Transcription.Provider,
		APIKey:   c.Transcription.APIKey,
		Language: c.Transcription.Language,
		Model:    c.Transcription.Model,
	}

	if config.APIKey == "" {
		config.APIKey = os.Getenv("OPENAI_API_KEY")
	}

	return config
}

func (c *Config) ToInjectionConfig() injection.Config {
	return injection.Config{
		Mode:             c.Injection.Mode,
		RestoreClipboard: c.Injection.RestoreClipboard,
		WtypeTimeout:     c.Injection.WtypeTimeout,
		ClipboardTimeout: c.Injection.ClipboardTimeout,
	}
}

func (c *Config) Validate() error {
	// Recording
	if c.Recording.SampleRate <= 0 {
		return fmt.Errorf("invalid recording.sample_rate: %d", c.Recording.SampleRate)
	}
	if c.Recording.Channels <= 0 {
		return fmt.Errorf("invalid recording.channels: %d", c.Recording.Channels)
	}
	if c.Recording.BufferSize <= 0 {
		return fmt.Errorf("invalid recording.buffer_size: %d", c.Recording.BufferSize)
	}
	if c.Recording.ChannelBufferSize <= 0 {
		return fmt.Errorf("invalid recording.channel_buffer_size: %d", c.Recording.ChannelBufferSize)
	}
	if c.Recording.Format == "" {
		return fmt.Errorf("invalid recording.format: empty")
	}
	if c.Recording.Timeout <= 0 {
		return fmt.Errorf("invalid recording.timeout: %v", c.Recording.Timeout)
	}

	// Transcription
	if c.Transcription.Provider == "" {
		return fmt.Errorf("invalid transcription.provider: empty")
	}
	if c.Transcription.Provider == "openai" {
		apiKey := c.Transcription.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			return fmt.Errorf("OpenAI API key required: not found in config (transcription.api_key) or environment variable (OPENAI_API_KEY)")
		}

		// Validate language code if provided (empty string means auto-detect)
		if c.Transcription.Language != "" && !isValidLanguageCode(c.Transcription.Language) {
			return fmt.Errorf("invalid transcription.language: %s (use empty string for auto-detect or ISO-639-1 codes like 'en', 'es', 'fr')", c.Transcription.Language)
		}
	}
	if c.Transcription.Model == "" {
		return fmt.Errorf("invalid transcription.model: empty")
	}

	// Injection
	validModes := map[string]bool{"clipboard": true, "type": true, "fallback": true}
	if !validModes[c.Injection.Mode] {
		return fmt.Errorf("invalid injection.mode: %s (must be clipboard, type, or fallback)", c.Injection.Mode)
	}
	if c.Injection.WtypeTimeout <= 0 {
		return fmt.Errorf("invalid injection.wtype_timeout: %v", c.Injection.WtypeTimeout)
	}
	if c.Injection.ClipboardTimeout <= 0 {
		return fmt.Errorf("invalid injection.clipboard_timeout: %v", c.Injection.ClipboardTimeout)
	}

	// Notifications
	validTypes := map[string]bool{"desktop": true, "log": true, "none": true}
	if !validTypes[c.Notifications.Type] {
		return fmt.Errorf("invalid notifications.type: %s (must be desktop, log, or none)", c.Notifications.Type)
	}

	return nil
}

func isValidLanguageCode(code string) bool {
	validCodes := map[string]bool{
		"en": true, "es": true, "fr": true, "de": true, "it": true, "pt": true,
		"ru": true, "ja": true, "ko": true, "zh": true, "ar": true, "hi": true,
		"nl": true, "sv": true, "da": true, "no": true, "fi": true, "pl": true,
		"tr": true, "he": true, "th": true, "vi": true, "id": true, "ms": true,
		"uk": true, "cs": true, "hu": true, "ro": true, "bg": true, "hr": true,
		"sk": true, "sl": true, "et": true, "lv": true, "lt": true, "mt": true,
		"cy": true, "ga": true, "eu": true, "ca": true, "gl": true, "is": true,
		"mk": true, "sq": true, "az": true, "be": true, "ka": true, "hy": true,
		"kk": true, "ky": true, "tg": true, "uz": true, "mn": true, "ne": true,
		"si": true, "km": true, "lo": true, "my": true, "fa": true, "ps": true,
		"ur": true, "bn": true, "ta": true, "te": true, "ml": true, "kn": true,
		"gu": true, "pa": true, "or": true, "as": true, "mr": true, "sa": true,
		"sw": true, "yo": true, "ig": true, "ha": true, "zu": true, "xh": true,
		"af": true, "am": true, "mg": true, "so": true, "sn": true, "rw": true,
	}
	return validCodes[code]
}

func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	hyprvoiceDir := filepath.Join(configDir, "hyprvoice")
	if err := os.MkdirAll(hyprvoiceDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(hyprvoiceDir, "config.toml"), nil
}

func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, create it with defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("Config: no config file found at %s, creating with defaults", configPath)
		if err := SaveDefaultConfig(); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		log.Printf("Config: default configuration created successfully")
		return Load() // Recursively load the config, now file will exist
	}

	log.Printf("Config: loading configuration from %s", configPath)
	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	log.Printf("Config: configuration loaded successfully")
	return &config, nil
}

func SaveDefaultConfig() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	configContent := `# Hyprvoice Configuration
# This file is automatically generated with defaults.
# Edit values as needed - changes are applied immediately without daemon restart.

# Audio Recording Configuration
[recording]
  sample_rate = 16000          # Audio sample rate in Hz (16000 recommended for speech)
  channels = 1                 # Number of audio channels (1 = mono, 2 = stereo)
  format = "s16"               # Audio format (s16 = 16-bit signed integers)
  buffer_size = 8192           # Internal buffer size in bytes (larger = less CPU, more latency)
  device = ""                  # PipeWire audio device (empty = use default microphone)
  channel_buffer_size = 30     # Audio frame buffer size (frames to buffer)
  timeout = "5m"               # Maximum recording duration (e.g., "30s", "2m", "5m")

# Speech Transcription Configuration  
[transcription]
  provider = "openai"          # Transcription service ("openai" only currently supported)
  api_key = ""                 # OpenAI API key (or set OPENAI_API_KEY environment variable)
  language = ""                # Language code (empty for auto-detect, "en", "it", "es", "fr", etc.)
  model = "whisper-1"          # OpenAI model name ("whisper-1" recommended)

# Text Injection Configuration
[injection]
  mode = "fallback"            # Injection method ("clipboard", "type", "fallback")
  restore_clipboard = true     # Restore original clipboard after injection
  wtype_timeout = "5s"         # Timeout for direct typing via wtype
  clipboard_timeout = "3s"     # Timeout for clipboard operations

# Desktop Notification Configuration
[notifications]
  enabled = true               # Enable desktop notifications
  type = "desktop"             # Notification type ("desktop", "log", "none")

# Mode explanations:
# - "clipboard": Copy text to clipboard only
# - "type": Direct typing via wtype only  
# - "fallback": Try typing first, fallback to clipboard if it fails
#
# Language codes: Use empty string ("") for automatic detection, or specific codes like:
# "en" (English), "it" (Italian), "es" (Spanish), "fr" (French), "de" (German), etc.
`

	if _, err := file.WriteString(configContent); err != nil {
		return fmt.Errorf("failed to write config content: %w", err)
	}

	return nil
}
