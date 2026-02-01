package config

import "fmt"

func (c *Config) Validate() error {
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

	if c.Transcription.Provider == "" {
		return fmt.Errorf("invalid transcription.provider: empty")
	}

	apiKey := c.resolveAPIKeyForProvider(c.Transcription.Provider)

	switch c.Transcription.Provider {
	case "openai":
		if apiKey == "" {
			return fmt.Errorf("OpenAI API key required: not found in config (providers.openai.api_key, transcription.api_key) or environment variable (OPENAI_API_KEY)")
		}

		if c.Transcription.Language != "" && !isValidLanguageCode(c.Transcription.Language) {
			return fmt.Errorf("invalid transcription.language: %s (use empty string for auto-detect or ISO-639-1 codes like 'en', 'es', 'fr')", c.Transcription.Language)
		}

	case "groq-transcription":
		if apiKey == "" {
			return fmt.Errorf("Groq API key required: not found in config (providers.groq.api_key, transcription.api_key) or environment variable (GROQ_API_KEY)")
		}

		if c.Transcription.Language != "" && !isValidLanguageCode(c.Transcription.Language) {
			return fmt.Errorf("invalid transcription.language: %s (use empty string for auto-detect or ISO-639-1 codes like 'en', 'es', 'fr')", c.Transcription.Language)
		}

		validGroqModels := map[string]bool{"whisper-large-v3": true, "whisper-large-v3-turbo": true}
		if c.Transcription.Model != "" && !validGroqModels[c.Transcription.Model] {
			return fmt.Errorf("invalid model for groq-transcription: %s (must be whisper-large-v3 or whisper-large-v3-turbo)", c.Transcription.Model)
		}

	case "groq-translation":
		if apiKey == "" {
			return fmt.Errorf("Groq API key required: not found in config (providers.groq.api_key, transcription.api_key) or environment variable (GROQ_API_KEY)")
		}

		if c.Transcription.Language != "" && !isValidLanguageCode(c.Transcription.Language) {
			return fmt.Errorf("invalid transcription.language: %s (use empty string for auto-detect or ISO-639-1 codes like 'en', 'es', 'fr')", c.Transcription.Language)
		}

		if c.Transcription.Model != "" && c.Transcription.Model != "whisper-large-v3" {
			return fmt.Errorf("invalid model for groq-translation: %s (must be whisper-large-v3, turbo version not supported for translation)", c.Transcription.Model)
		}

	case "mistral-transcription":
		if apiKey == "" {
			return fmt.Errorf("Mistral API key required: not found in config (providers.mistral.api_key, transcription.api_key) or environment variable (MISTRAL_API_KEY)")
		}

		if c.Transcription.Language != "" && !isValidLanguageCode(c.Transcription.Language) {
			return fmt.Errorf("invalid transcription.language: %s (use empty string for auto-detect or ISO-639-1 codes like 'en', 'es', 'fr')", c.Transcription.Language)
		}

		validMistralModels := map[string]bool{"voxtral-mini-latest": true, "voxtral-mini-2507": true}
		if c.Transcription.Model != "" && !validMistralModels[c.Transcription.Model] {
			return fmt.Errorf("invalid model for mistral-transcription: %s (must be voxtral-mini-latest or voxtral-mini-2507)", c.Transcription.Model)
		}

	case "elevenlabs":
		if apiKey == "" {
			return fmt.Errorf("ElevenLabs API key required: not found in config (providers.elevenlabs.api_key, transcription.api_key) or environment variable (ELEVENLABS_API_KEY)")
		}

		if c.Transcription.Language != "" && !isValidLanguageCode(c.Transcription.Language) {
			return fmt.Errorf("invalid transcription.language: %s (use empty string for auto-detect or ISO-639-1 codes like 'en', 'pt', 'es')", c.Transcription.Language)
		}

		validModels := map[string]bool{"scribe_v1": true, "scribe_v2": true}
		if c.Transcription.Model != "" && !validModels[c.Transcription.Model] {
			return fmt.Errorf("invalid model for elevenlabs: %s (must be scribe_v1 or scribe_v2)", c.Transcription.Model)
		}

	case "whisper-cpp":
		// whisper-cpp is local, no API key required
		if c.Transcription.Language != "" && !isValidLanguageCode(c.Transcription.Language) {
			return fmt.Errorf("invalid transcription.language: %s (use empty string for auto-detect or ISO-639-1 codes like 'en', 'es', 'fr')", c.Transcription.Language)
		}

		validWhisperModels := map[string]bool{
			"tiny.en": true, "base.en": true, "small.en": true, "medium.en": true,
			"tiny": true, "base": true, "small": true, "medium": true, "large-v3": true,
		}
		if c.Transcription.Model != "" && !validWhisperModels[c.Transcription.Model] {
			return fmt.Errorf("invalid model for whisper-cpp: %s (must be tiny.en, base.en, small.en, medium.en, tiny, base, small, medium, or large-v3)", c.Transcription.Model)
		}

	default:
		return fmt.Errorf("unsupported transcription.provider: %s (must be openai, groq-transcription, groq-translation, mistral-transcription, elevenlabs, or whisper-cpp)", c.Transcription.Provider)
	}

	if c.Transcription.Model == "" {
		return fmt.Errorf("invalid transcription.model: empty")
	}

	if c.LLM.Enabled {
		if c.LLM.Provider == "" {
			return fmt.Errorf("llm.provider required when llm.enabled = true")
		}
		if c.LLM.Model == "" {
			return fmt.Errorf("llm.model required when llm.enabled = true")
		}

		validLLMProviders := map[string]bool{"openai": true, "groq": true}
		if !validLLMProviders[c.LLM.Provider] {
			return fmt.Errorf("invalid llm.provider: %s (must be openai or groq)", c.LLM.Provider)
		}

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
