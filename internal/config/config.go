package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/leonardotrapani/hyprvoice/internal/injection"
	"github.com/leonardotrapani/hyprvoice/internal/notify"
	"github.com/leonardotrapani/hyprvoice/internal/recording"
	"github.com/leonardotrapani/hyprvoice/internal/transcriber"
)

type Config struct {
	Recording     RecordingConfig           `toml:"recording"`
	Transcription TranscriptionConfig       `toml:"transcription"`
	Injection     InjectionConfig           `toml:"injection"`
	Notifications NotificationsConfig       `toml:"notifications"`
	Providers     map[string]ProviderConfig `toml:"providers"`
	Keywords      []string                  `toml:"keywords"`
	LLM           LLMConfig                 `toml:"llm"`
}

// ProviderConfig holds API key for a provider
type ProviderConfig struct {
	APIKey string `toml:"api_key"`
}

// LLMConfig configures the LLM post-processing phase
type LLMConfig struct {
	Enabled        bool                    `toml:"enabled"`
	Provider       string                  `toml:"provider"`
	Model          string                  `toml:"model"`
	PostProcessing LLMPostProcessingConfig `toml:"post_processing"`
	CustomPrompt   LLMCustomPromptConfig   `toml:"custom_prompt"`
}

// LLMPostProcessingConfig controls text cleanup options
type LLMPostProcessingConfig struct {
	RemoveStutters    bool `toml:"remove_stutters"`
	AddPunctuation    bool `toml:"add_punctuation"`
	FixGrammar        bool `toml:"fix_grammar"`
	RemoveFillerWords bool `toml:"remove_filler_words"`
}

// LLMCustomPromptConfig allows custom prompts
type LLMCustomPromptConfig struct {
	Enabled bool   `toml:"enabled"`
	Prompt  string `toml:"prompt"`
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
	Backends         []string      `toml:"backends"`
	YdotoolTimeout   time.Duration `toml:"ydotool_timeout"`
	WtypeTimeout     time.Duration `toml:"wtype_timeout"`
	ClipboardTimeout time.Duration `toml:"clipboard_timeout"`
}

type NotificationsConfig struct {
	Enabled  bool           `toml:"enabled"`
	Type     string         `toml:"type"` // "desktop", "log", "none"
	Messages MessagesConfig `toml:"messages"`
}

type MessageConfig struct {
	Title string `toml:"title"`
	Body  string `toml:"body"`
}

type MessagesConfig struct {
	RecordingStarted   MessageConfig `toml:"recording_started"`
	Transcribing       MessageConfig `toml:"transcribing"`
	ConfigReloaded     MessageConfig `toml:"config_reloaded"`
	OperationCancelled MessageConfig `toml:"operation_cancelled"`
	RecordingAborted   MessageConfig `toml:"recording_aborted"`
	InjectionAborted   MessageConfig `toml:"injection_aborted"`
}

// Resolve merges user config with defaults from MessageDefs
func (m *MessagesConfig) Resolve() map[notify.MessageType]notify.Message {
	result := make(map[notify.MessageType]notify.Message)

	// Build toml tag → field index map
	v := reflect.ValueOf(m).Elem()
	t := v.Type()
	tagToField := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		tagToField[t.Field(i).Tag.Get("toml")] = i
	}

	for _, def := range notify.MessageDefs {
		msg := notify.Message{
			Title:   def.DefaultTitle,
			Body:    def.DefaultBody,
			IsError: def.IsError,
		}
		if idx, ok := tagToField[def.ConfigKey]; ok {
			userMsg := v.Field(idx).Interface().(MessageConfig)
			if userMsg.Title != "" {
				msg.Title = userMsg.Title
			}
			if userMsg.Body != "" {
				msg.Body = userMsg.Body
			}
		}
		result[def.Type] = msg
	}
	return result
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
		Language: c.Transcription.Language,
		Model:    c.Transcription.Model,
	}

	// Resolve API key: providers map -> legacy transcription.api_key -> environment variable
	config.APIKey = c.resolveAPIKeyForProvider(c.Transcription.Provider)

	return config
}

// resolveAPIKeyForProvider returns the API key for a provider from multiple sources
func (c *Config) resolveAPIKeyForProvider(provider string) string {
	// Map transcription provider names to provider registry names
	providerName := provider
	envVar := ""
	switch provider {
	case "openai":
		providerName = "openai"
		envVar = "OPENAI_API_KEY"
	case "groq-transcription", "groq-translation":
		providerName = "groq"
		envVar = "GROQ_API_KEY"
	case "mistral-transcription":
		providerName = "mistral"
		envVar = "MISTRAL_API_KEY"
	case "elevenlabs":
		providerName = "elevenlabs"
		envVar = "ELEVENLABS_API_KEY"
	}

	// 1. Check providers map
	if c.Providers != nil {
		if pc, ok := c.Providers[providerName]; ok && pc.APIKey != "" {
			return pc.APIKey
		}
	}

	// 2. Check legacy transcription.api_key (backward compatibility)
	if c.Transcription.APIKey != "" {
		return c.Transcription.APIKey
	}

	// 3. Check environment variable
	if envVar != "" {
		return os.Getenv(envVar)
	}

	return ""
}

// LLMAdapterConfig is the configuration passed to the LLM adapter
type LLMAdapterConfig struct {
	Provider          string
	APIKey            string
	Model             string
	RemoveStutters    bool
	AddPunctuation    bool
	FixGrammar        bool
	RemoveFillerWords bool
	CustomPrompt      string
	Keywords          []string
}

// ToLLMConfig returns the LLM adapter configuration
func (c *Config) ToLLMConfig() LLMAdapterConfig {
	config := LLMAdapterConfig{
		Provider:          c.LLM.Provider,
		Model:             c.LLM.Model,
		RemoveStutters:    c.LLM.PostProcessing.RemoveStutters,
		AddPunctuation:    c.LLM.PostProcessing.AddPunctuation,
		FixGrammar:        c.LLM.PostProcessing.FixGrammar,
		RemoveFillerWords: c.LLM.PostProcessing.RemoveFillerWords,
		Keywords:          c.Keywords,
	}

	// Resolve API key for LLM provider
	if c.LLM.Provider != "" {
		config.APIKey = c.resolveAPIKeyForLLMProvider(c.LLM.Provider)
	}

	// Add custom prompt if enabled
	if c.LLM.CustomPrompt.Enabled && c.LLM.CustomPrompt.Prompt != "" {
		config.CustomPrompt = c.LLM.CustomPrompt.Prompt
	}

	return config
}

// resolveAPIKeyForLLMProvider returns the API key for an LLM provider
func (c *Config) resolveAPIKeyForLLMProvider(provider string) string {
	envVar := ""
	switch provider {
	case "openai":
		envVar = "OPENAI_API_KEY"
	case "groq":
		envVar = "GROQ_API_KEY"
	}

	// 1. Check providers map
	if c.Providers != nil {
		if pc, ok := c.Providers[provider]; ok && pc.APIKey != "" {
			return pc.APIKey
		}
	}

	// 2. Check environment variable
	if envVar != "" {
		return os.Getenv(envVar)
	}

	return ""
}

// IsLLMEnabled returns true if LLM post-processing is enabled and configured
func (c *Config) IsLLMEnabled() bool {
	return c.LLM.Enabled && c.LLM.Provider != "" && c.LLM.Model != ""
}

func (c *Config) ToInjectionConfig() injection.Config {
	return injection.Config{
		Backends:         c.Injection.Backends,
		YdotoolTimeout:   c.Injection.YdotoolTimeout,
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

	// Validate provider-specific settings using unified API key resolution
	apiKey := c.resolveAPIKeyForProvider(c.Transcription.Provider)

	switch c.Transcription.Provider {
	case "openai":
		if apiKey == "" {
			return fmt.Errorf("OpenAI API key required: not found in config (providers.openai.api_key, transcription.api_key) or environment variable (OPENAI_API_KEY)")
		}

		// Validate language code if provided (empty string means auto-detect)
		if c.Transcription.Language != "" && !isValidLanguageCode(c.Transcription.Language) {
			return fmt.Errorf("invalid transcription.language: %s (use empty string for auto-detect or ISO-639-1 codes like 'en', 'es', 'fr')", c.Transcription.Language)
		}

	case "groq-transcription":
		if apiKey == "" {
			return fmt.Errorf("Groq API key required: not found in config (providers.groq.api_key, transcription.api_key) or environment variable (GROQ_API_KEY)")
		}

		// Validate language code if provided (empty string means auto-detect)
		if c.Transcription.Language != "" && !isValidLanguageCode(c.Transcription.Language) {
			return fmt.Errorf("invalid transcription.language: %s (use empty string for auto-detect or ISO-639-1 codes like 'en', 'es', 'fr')", c.Transcription.Language)
		}

		// Validate Groq model
		validGroqModels := map[string]bool{"whisper-large-v3": true, "whisper-large-v3-turbo": true}
		if c.Transcription.Model != "" && !validGroqModels[c.Transcription.Model] {
			return fmt.Errorf("invalid model for groq-transcription: %s (must be whisper-large-v3 or whisper-large-v3-turbo)", c.Transcription.Model)
		}

	case "groq-translation":
		if apiKey == "" {
			return fmt.Errorf("Groq API key required: not found in config (providers.groq.api_key, transcription.api_key) or environment variable (GROQ_API_KEY)")
		}

		// For translation, language field hints at source language (output is always English)
		if c.Transcription.Language != "" && !isValidLanguageCode(c.Transcription.Language) {
			return fmt.Errorf("invalid transcription.language: %s (use empty string for auto-detect or ISO-639-1 codes like 'en', 'es', 'fr')", c.Transcription.Language)
		}

		// Validate Groq translation model - only whisper-large-v3 is supported (no turbo)
		if c.Transcription.Model != "" && c.Transcription.Model != "whisper-large-v3" {
			return fmt.Errorf("invalid model for groq-translation: %s (must be whisper-large-v3, turbo version not supported for translation)", c.Transcription.Model)
		}

	case "mistral-transcription":
		if apiKey == "" {
			return fmt.Errorf("Mistral API key required: not found in config (providers.mistral.api_key, transcription.api_key) or environment variable (MISTRAL_API_KEY)")
		}

		// Validate language code if provided (empty string means auto-detect)
		if c.Transcription.Language != "" && !isValidLanguageCode(c.Transcription.Language) {
			return fmt.Errorf("invalid transcription.language: %s (use empty string for auto-detect or ISO-639-1 codes like 'en', 'es', 'fr')", c.Transcription.Language)
		}

		// Validate Mistral model
		validMistralModels := map[string]bool{"voxtral-mini-latest": true, "voxtral-mini-2507": true}
		if c.Transcription.Model != "" && !validMistralModels[c.Transcription.Model] {
			return fmt.Errorf("invalid model for mistral-transcription: %s (must be voxtral-mini-latest or voxtral-mini-2507)", c.Transcription.Model)
		}

	case "elevenlabs":
		if apiKey == "" {
			return fmt.Errorf("ElevenLabs API key required: not found in config (providers.elevenlabs.api_key, transcription.api_key) or environment variable (ELEVENLABS_API_KEY)")
		}

		// Validate language code if provided (empty string means auto-detect)
		if c.Transcription.Language != "" && !isValidLanguageCode(c.Transcription.Language) {
			return fmt.Errorf("invalid transcription.language: %s (use empty string for auto-detect or ISO-639-1 codes like 'en', 'pt', 'es')", c.Transcription.Language)
		}

		// Validate Eleven Labs model
		validModels := map[string]bool{"scribe_v1": true, "scribe_v2": true}
		if c.Transcription.Model != "" && !validModels[c.Transcription.Model] {
			return fmt.Errorf("invalid model for elevenlabs: %s (must be scribe_v1 or scribe_v2)", c.Transcription.Model)
		}

	default:
		return fmt.Errorf("unsupported transcription.provider: %s (must be openai, groq-transcription, groq-translation, mistral-transcription, or elevenlabs)", c.Transcription.Provider)
	}

	if c.Transcription.Model == "" {
		return fmt.Errorf("invalid transcription.model: empty")
	}

	// LLM (only validate if enabled)
	if c.LLM.Enabled {
		if c.LLM.Provider == "" {
			return fmt.Errorf("llm.provider required when llm.enabled = true")
		}
		if c.LLM.Model == "" {
			return fmt.Errorf("llm.model required when llm.enabled = true")
		}

		// Validate LLM provider
		validLLMProviders := map[string]bool{"openai": true, "groq": true}
		if !validLLMProviders[c.LLM.Provider] {
			return fmt.Errorf("invalid llm.provider: %s (must be openai or groq)", c.LLM.Provider)
		}

		// Check API key for LLM provider
		llmAPIKey := c.resolveAPIKeyForLLMProvider(c.LLM.Provider)
		if llmAPIKey == "" {
			switch c.LLM.Provider {
			case "openai":
				return fmt.Errorf("OpenAI API key required for LLM: not found in config (providers.openai.api_key) or environment variable (OPENAI_API_KEY)")
			case "groq":
				return fmt.Errorf("Groq API key required for LLM: not found in config (providers.groq.api_key) or environment variable (GROQ_API_KEY)")
			}
		}
	}

	// Injection
	if len(c.Injection.Backends) == 0 {
		return fmt.Errorf("invalid injection.backends: empty (must have at least one backend)")
	}
	validBackends := map[string]bool{"ydotool": true, "wtype": true, "clipboard": true}
	for _, backend := range c.Injection.Backends {
		if !validBackends[backend] {
			return fmt.Errorf("invalid injection.backends: unknown backend %q (must be ydotool, wtype, or clipboard)", backend)
		}
	}
	if c.Injection.YdotoolTimeout <= 0 {
		return fmt.Errorf("invalid injection.ydotool_timeout: %v", c.Injection.YdotoolTimeout)
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

// legacyInjectionConfig for migration from old mode-based config
type legacyInjectionConfig struct {
	Mode string `toml:"mode"`
}

// legacyTranscriptionConfig for migration from old api_key in transcription
type legacyTranscriptionConfig struct {
	APIKey string `toml:"api_key"`
}

type legacyConfig struct {
	Injection     legacyInjectionConfig     `toml:"injection"`
	Transcription legacyTranscriptionConfig `toml:"transcription"`
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

	// Parse legacy config for migrations
	var legacy legacyConfig
	toml.DecodeFile(configPath, &legacy)

	// Migrate legacy mode-based config to backends
	if len(config.Injection.Backends) == 0 {
		config.migrateInjectionMode(legacy.Injection.Mode)
	}

	// Migrate legacy transcription.api_key to providers map
	if legacy.Transcription.APIKey != "" && config.Providers == nil {
		config.migrateTranscriptionAPIKey(legacy.Transcription.APIKey)
	}

	// Initialize providers map if nil
	if config.Providers == nil {
		config.Providers = make(map[string]ProviderConfig)
	}

	// Set LLM defaults if not configured
	config.applyLLMDefaults()

	log.Printf("Config: configuration loaded successfully")
	return &config, nil
}

// migrateTranscriptionAPIKey migrates old transcription.api_key to providers map
func (c *Config) migrateTranscriptionAPIKey(apiKey string) {
	if c.Providers == nil {
		c.Providers = make(map[string]ProviderConfig)
	}

	// Determine which provider this key is for based on transcription.provider
	providerName := c.Transcription.Provider
	switch providerName {
	case "openai":
		c.Providers["openai"] = ProviderConfig{APIKey: apiKey}
	case "groq-transcription", "groq-translation":
		c.Providers["groq"] = ProviderConfig{APIKey: apiKey}
	case "mistral-transcription":
		c.Providers["mistral"] = ProviderConfig{APIKey: apiKey}
	case "elevenlabs":
		c.Providers["elevenlabs"] = ProviderConfig{APIKey: apiKey}
	default:
		// Unknown provider, try to guess based on key prefix
		if len(apiKey) > 3 && apiKey[:3] == "sk-" {
			c.Providers["openai"] = ProviderConfig{APIKey: apiKey}
		} else if len(apiKey) > 4 && apiKey[:4] == "gsk_" {
			c.Providers["groq"] = ProviderConfig{APIKey: apiKey}
		}
	}

	log.Printf("Config: migrated transcription.api_key to providers map. Run 'hyprvoice configure' to update config format.")
}

// applyLLMDefaults sets default values for LLM config
func (c *Config) applyLLMDefaults() {
	// Default post-processing options to true if LLM is enabled and not explicitly set
	// We detect "not set" by checking if all booleans are false (zero value)
	// Since the default behavior should be all true, we only apply if everything is false
	pp := &c.LLM.PostProcessing
	if !pp.RemoveStutters && !pp.AddPunctuation && !pp.FixGrammar && !pp.RemoveFillerWords {
		// Nothing was set, apply defaults
		pp.RemoveStutters = true
		pp.AddPunctuation = true
		pp.FixGrammar = true
		pp.RemoveFillerWords = true
	}
}

// migrateInjectionMode converts old mode field to new backends array
func (c *Config) migrateInjectionMode(mode string) {
	switch mode {
	case "clipboard":
		c.Injection.Backends = []string{"clipboard"}
		log.Printf("Config: migrated injection.mode='clipboard' to backends=['clipboard']")
	case "type":
		c.Injection.Backends = []string{"wtype"}
		log.Printf("Config: migrated injection.mode='type' to backends=['wtype']")
	case "fallback":
		c.Injection.Backends = []string{"wtype", "clipboard"}
		log.Printf("Config: migrated injection.mode='fallback' to backends=['wtype', 'clipboard']")
	default:
		// Default for new installs or unknown modes
		c.Injection.Backends = []string{"ydotool", "wtype", "clipboard"}
		if mode != "" {
			log.Printf("Config: unknown injection.mode='%s', using default backends", mode)
		}
	}

	// Set default ydotool timeout if not set
	if c.Injection.YdotoolTimeout == 0 {
		c.Injection.YdotoolTimeout = 5 * time.Second
	}

	log.Printf("Config: legacy 'mode' config detected - please update your config.toml to use 'backends' instead")
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
  provider = "openai"          # Transcription service: "openai", "groq-transcription", "groq-translation", "mistral-transcription", or "elevenlabs"
  api_key = ""                 # API key (or set OPENAI_API_KEY/GROQ_API_KEY/MISTRAL_API_KEY/ELEVENLABS_API_KEY environment variable)
  language = ""                # Language code (empty for auto-detect, "en", "it", "es", "fr", etc.)
  model = "whisper-1"          # Model: OpenAI="whisper-1", Groq="whisper-large-v3", Mistral="voxtral-mini-latest", ElevenLabs="scribe_v1"

# Text Injection Configuration
[injection]
  backends = ["ydotool", "wtype", "clipboard"]  # Ordered fallback chain (tries each until one succeeds)
  ydotool_timeout = "5s"       # Timeout for ydotool commands
  wtype_timeout = "5s"         # Timeout for wtype commands
  clipboard_timeout = "3s"     # Timeout for clipboard operations

# Desktop Notification Configuration
[notifications]
  enabled = true               # Enable desktop notifications
  type = "desktop"             # Notification type ("desktop", "log", "none")

  # Custom notification messages (optional - defaults shown below)
  # Uncomment and modify to customize notification text
  # [notifications.messages]
  #   [notifications.messages.recording_started]
  #     title = "Hyprvoice"
  #     body = "Recording Started"
  #   [notifications.messages.transcribing]
  #     title = "Hyprvoice"
  #     body = "Recording Ended... Transcribing"
  #   [notifications.messages.config_reloaded]
  #     title = "Hyprvoice"
  #     body = "Config Reloaded"
  #   [notifications.messages.operation_cancelled]
  #     title = "Hyprvoice"
  #     body = "Operation Cancelled"
  #   [notifications.messages.recording_aborted]
  #     body = "Recording Aborted"
  #   [notifications.messages.injection_aborted]
  #     body = "Injection Aborted"
  #
  # Emoji-only example (for minimal pill-style notifications):
  #   [notifications.messages.recording_started]
  #     title = ""
  #     body = "🎤"
  #   [notifications.messages.transcribing]
  #     title = ""
  #     body = "⏳"
  #   [notifications.messages.config_reloaded]
  #     title = ""
  #     body = "🔧"

# Backend explanations:
# - "ydotool": Uses ydotool (requires ydotoold daemon running for ydotool v1.0.0+). Most compatible with Chromium/Electron apps.
# - "wtype": Uses wtype for Wayland. May have issues with some Chromium-based apps.
# - "clipboard": Copies text to clipboard only (most reliable, but requires manual paste).
#
# The backends are tried in order. First successful one wins.
# Example configurations:
#   backends = ["clipboard"]                      # Clipboard only (safest)
#   backends = ["wtype", "clipboard"]             # wtype with clipboard fallback
#   backends = ["ydotool", "wtype", "clipboard"]  # Full fallback chain (default)
#
# Provider explanations:
# - "openai": OpenAI Whisper API (cloud-based, requires OPENAI_API_KEY)
# - "groq-transcription": Groq Whisper API for transcription (fast, requires GROQ_API_KEY)
#     Models: whisper-large-v3 or whisper-large-v3-turbo
# - "groq-translation": Groq Whisper API for translation to English (always outputs English text)
#     Models: whisper-large-v3 only (turbo not supported for translation)
# - "mistral-transcription": Mistral Voxtral API (excellent for European languages, requires MISTRAL_API_KEY)
#     Models: voxtral-mini-latest or voxtral-mini-2507
# - "elevenlabs": ElevenLabs Scribe API (excellent accuracy, 99 languages, requires ELEVENLABS_API_KEY)
#     Models: scribe_v1 (99 languages, best accuracy) or scribe_v2 (90 languages, real-time)
#
# Language codes: Use empty string ("") for automatic detection, or specific codes like:
# "en" (English), "it" (Italian), "es" (Spanish), "fr" (French), "de" (German), etc.
# For groq-translation, the language field hints at the source audio language for better accuracy.
`

	if _, err := file.WriteString(configContent); err != nil {
		return fmt.Errorf("failed to write config content: %w", err)
	}

	return nil
}
