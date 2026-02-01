package config

import (
	"os"

	"github.com/leonardotrapani/hyprvoice/internal/injection"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
	"github.com/leonardotrapani/hyprvoice/internal/recording"
	"github.com/leonardotrapani/hyprvoice/internal/transcriber"
)

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
		Provider:  c.Transcription.Provider,
		Language:  c.resolveEffectiveLanguage(),
		Model:     c.Transcription.Model,
		Keywords:  c.Keywords,
		Threads:   c.Transcription.Threads,
		Streaming: c.Transcription.Streaming,
	}

	config.APIKey = c.resolveAPIKeyForProvider(c.Transcription.Provider)

	return config
}

// resolveEffectiveLanguage returns the effective language for transcription.
// transcription.language overrides general.language if set.
func (c *Config) resolveEffectiveLanguage() string {
	if c.Transcription.Language != "" {
		return c.Transcription.Language
	}
	return c.General.Language
}

// resolveAPIKeyForProvider returns the API key for a provider from multiple sources
func (c *Config) resolveAPIKeyForProvider(providerName string) string {
	baseName := provider.BaseProviderName(providerName)
	envVar := provider.EnvVarForProvider(providerName)

	if c.Providers != nil {
		if pc, ok := c.Providers[baseName]; ok && pc.APIKey != "" {
			return pc.APIKey
		}
	}

	if c.Transcription.APIKey != "" {
		return c.Transcription.APIKey
	}

	if envVar != "" {
		return os.Getenv(envVar)
	}

	return ""
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

	if c.LLM.Provider != "" {
		config.APIKey = c.resolveAPIKeyForLLMProvider(c.LLM.Provider)
	}

	if c.LLM.CustomPrompt.Enabled && c.LLM.CustomPrompt.Prompt != "" {
		config.CustomPrompt = c.LLM.CustomPrompt.Prompt
	}

	return config
}

// resolveAPIKeyForLLMProvider returns the API key for an LLM provider
func (c *Config) resolveAPIKeyForLLMProvider(providerName string) string {
	envVar := provider.EnvVarForProvider(providerName)

	if c.Providers != nil {
		if pc, ok := c.Providers[providerName]; ok && pc.APIKey != "" {
			return pc.APIKey
		}
	}

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
