package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"
)

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

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("Config: no config file found at %s, creating with defaults", configPath)
		if err := SaveDefaultConfig(); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		log.Printf("Config: default configuration created successfully")
		return Load()
	}

	log.Printf("Config: loading configuration from %s", configPath)
	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	var legacy legacyConfig
	toml.DecodeFile(configPath, &legacy)

	if len(config.Injection.Backends) == 0 {
		config.migrateInjectionMode(legacy.Injection.Mode)
	}

	if legacy.Transcription.APIKey != "" && config.Providers == nil {
		config.migrateTranscriptionAPIKey(legacy.Transcription.APIKey)
	}

	if config.Providers == nil {
		config.Providers = make(map[string]ProviderConfig)
	}

	config.applyLLMDefaults()
	config.applyThreadsDefault()
	config.migrateLanguageToGeneral()

	log.Printf("Config: configuration loaded successfully")
	return &config, nil
}

// applyThreadsDefault sets default threads for local transcription if not explicitly set
func (c *Config) applyThreadsDefault() {
	if c.Transcription.Threads == 0 {
		threads := runtime.NumCPU() - 1
		if threads < 1 {
			threads = 1
		}
		c.Transcription.Threads = threads
	}
}

// migrateTranscriptionAPIKey migrates old transcription.api_key to providers map
func (c *Config) migrateTranscriptionAPIKey(apiKey string) {
	if c.Providers == nil {
		c.Providers = make(map[string]ProviderConfig)
	}

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
	pp := &c.LLM.PostProcessing
	if !pp.RemoveStutters && !pp.AddPunctuation && !pp.FixGrammar && !pp.RemoveFillerWords {
		pp.RemoveStutters = true
		pp.AddPunctuation = true
		pp.FixGrammar = true
		pp.RemoveFillerWords = true
	}
}

// migrateLanguageToGeneral migrates old transcription.language to general.language
func (c *Config) migrateLanguageToGeneral() {
	if c.Transcription.Language != "" && c.General.Language == "" {
		c.General.Language = c.Transcription.Language
		log.Printf("Config: migrated language setting to [general] section")
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
		c.Injection.Backends = []string{"ydotool", "wtype", "clipboard"}
		if mode != "" {
			log.Printf("Config: unknown injection.mode='%s', using default backends", mode)
		}
	}

	if c.Injection.YdotoolTimeout == 0 {
		c.Injection.YdotoolTimeout = 5 * time.Second
	}

	log.Printf("Config: legacy 'mode' config detected - please update your config.toml to use 'backends' instead")
}
