package provider

// Provider name constants for config and registry
const (
	ProviderOpenAI     = "openai"
	ProviderGroq       = "groq"
	ProviderMistral    = "mistral"
	ProviderElevenLabs = "elevenlabs"
	ProviderDeepgram   = "deepgram"
	ProviderWhisperCpp = "whisper-cpp"
)

// Config provider names (used in config file transcription.provider)
const (
	ConfigProviderOpenAI               = "openai"
	ConfigProviderGroqTranscription    = "groq-transcription"
	ConfigProviderMistralTranscription = "mistral-transcription"
	ConfigProviderElevenLabs           = "elevenlabs"
	ConfigProviderDeepgram             = "deepgram"
	ConfigProviderWhisperCpp           = "whisper-cpp"
)

// Environment variable names for API keys
const (
	EnvOpenAIKey     = "OPENAI_API_KEY"
	EnvGroqKey       = "GROQ_API_KEY"
	EnvMistralKey    = "MISTRAL_API_KEY"
	EnvElevenLabsKey = "ELEVENLABS_API_KEY"
	EnvDeepgramKey   = "DEEPGRAM_API_KEY"
)

// Adapter type constants for transcription backends
const (
	AdapterOpenAI           = "openai"
	AdapterElevenLabs       = "elevenlabs"
	AdapterElevenLabsStream = "elevenlabs-streaming"
	AdapterDeepgram         = "deepgram"
	AdapterWhisperCpp       = "whisper-cpp"
	AdapterOpenAIRealtime   = "openai-realtime"
)

// BaseProviderName maps config provider names to registry provider names
// e.g. "groq-transcription" -> "groq", "mistral-transcription" -> "mistral"
func BaseProviderName(configProvider string) string {
	switch configProvider {
	case ConfigProviderGroqTranscription:
		return ProviderGroq
	case ConfigProviderMistralTranscription:
		return ProviderMistral
	default:
		return configProvider
	}
}

// EnvVarForProvider returns the environment variable name for a provider's API key
func EnvVarForProvider(provider string) string {
	base := BaseProviderName(provider)
	switch base {
	case ProviderOpenAI:
		return EnvOpenAIKey
	case ProviderGroq:
		return EnvGroqKey
	case ProviderMistral:
		return EnvMistralKey
	case ProviderElevenLabs:
		return EnvElevenLabsKey
	case ProviderDeepgram:
		return EnvDeepgramKey
	default:
		return ""
	}
}
