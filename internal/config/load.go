package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

var ErrConfigNotFound = errors.New("config not found")

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

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: run hyprvoice onboarding", ErrConfigNotFound)
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat config file %s: %w", configPath, err)
	}

	log.Printf("Config: loading configuration from %s", configPath)
	var config Config
	meta, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}
	if isLegacyConfig(meta, &config) {
		log.Printf("Config: legacy configuration detected - run hyprvoice onboarding")
		return nil, fmt.Errorf("%w: run hyprvoice onboarding", ErrConfigNotFound)
	}

	if config.Providers == nil {
		config.Providers = make(map[string]ProviderConfig)
	}

	config.applyLLMDefaults()
	config.applyThreadsDefault()

	log.Printf("Config: configuration loaded successfully")
	return &config, nil
}

func isLegacyConfig(meta toml.MetaData, config *Config) bool {
	if meta.IsDefined("transcription", "api_key") {
		return true
	}
	if meta.IsDefined("injection", "mode") {
		return true
	}
	if meta.IsDefined("general", "language") {
		return true
	}
	if config.Transcription.Provider == "groq-translation" {
		return true
	}
	return false
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
