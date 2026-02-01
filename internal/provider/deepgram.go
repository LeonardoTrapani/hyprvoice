package provider

// DeepgramProvider implements Provider for Deepgram transcription services
type DeepgramProvider struct{}

func (p *DeepgramProvider) Name() string {
	return "deepgram"
}

func (p *DeepgramProvider) RequiresAPIKey() bool {
	return true
}

func (p *DeepgramProvider) ValidateAPIKey(key string) bool {
	// Deepgram API keys are alphanumeric, just check non-empty
	return len(key) > 0
}

func (p *DeepgramProvider) IsLocal() bool {
	return false
}

func (p *DeepgramProvider) Models() []Model {
	// Nova-3 language support - maps to our 57 language list
	// from https://developers.deepgram.com/docs/models-languages-overview
	nova3Langs := []string{
		"ar", "be", "bs", "bg", "ca", "hr", "cs", "da", "nl", "en", "et", "fi",
		"fr", "de", "el", "hi", "hu", "id", "it", "ja", "kn", "ko", "lv", "lt",
		"mk", "ms", "mr", "no", "pl", "pt", "ro", "ru", "sr", "sk", "sl", "es",
		"sv", "tl", "ta", "tr", "uk", "vi",
	}

	// Nova-2 language support - subset of nova-3
	nova2Langs := []string{
		"bg", "ca", "zh", "cs", "da", "nl", "en", "et", "fi", "fr", "de", "el",
		"hi", "hu", "id", "it", "ja", "ko", "lv", "lt", "ms", "no", "pl", "pt",
		"ro", "ru", "sk", "es", "sv", "th", "tr", "uk", "vi",
	}

	return []Model{
		{
			ID:                 "nova-3",
			Name:               "Nova-3",
			Description:        "Best accuracy, 40+ languages, real-time",
			Type:               Transcription,
			Streaming:          true,
			Local:              false,
			AdapterType:        "deepgram",
			SupportedLanguages: nova3Langs,
			Endpoint:           &EndpointConfig{BaseURL: "wss://api.deepgram.com", Path: "/v1/listen"},
		},
		{
			ID:                 "nova-3-general",
			Name:               "Nova-3 General",
			Description:        "General purpose, same as nova-3",
			Type:               Transcription,
			Streaming:          true,
			Local:              false,
			AdapterType:        "deepgram",
			SupportedLanguages: nova3Langs,
			Endpoint:           &EndpointConfig{BaseURL: "wss://api.deepgram.com", Path: "/v1/listen"},
		},
		{
			ID:                 "nova-2",
			Name:               "Nova-2",
			Description:        "Fast, 30+ languages, filler words",
			Type:               Transcription,
			Streaming:          true,
			Local:              false,
			AdapterType:        "deepgram",
			SupportedLanguages: nova2Langs,
			Endpoint:           &EndpointConfig{BaseURL: "wss://api.deepgram.com", Path: "/v1/listen"},
		},
		{
			ID:                 "nova-2-general",
			Name:               "Nova-2 General",
			Description:        "General purpose, same as nova-2",
			Type:               Transcription,
			Streaming:          true,
			Local:              false,
			AdapterType:        "deepgram",
			SupportedLanguages: nova2Langs,
			Endpoint:           &EndpointConfig{BaseURL: "wss://api.deepgram.com", Path: "/v1/listen"},
		},
	}
}

func (p *DeepgramProvider) DefaultModel(t ModelType) string {
	switch t {
	case Transcription:
		return "nova-3"
	}
	return ""
}
