package provider

// ModelType represents the type of a model
type ModelType int

const (
	Transcription ModelType = iota
	LLM
)

// Model represents a model with full metadata
type Model struct {
	ID                 string          // unique identifier (e.g., "whisper-1", "gpt-4o-mini")
	Name               string          // display name (e.g., "Whisper 1", "GPT-4o Mini")
	Description        string          // short description
	Type               ModelType       // transcription or LLM
	SupportsBatch      bool            // can do batch/non-streaming transcription
	SupportsStreaming  bool            // can do real-time streaming transcription
	Local              bool            // runs locally (no API call)
	AdapterType        string          // which adapter to use (e.g., "openai", "elevenlabs", "whisper-cpp")
	StreamingAdapter   string          // adapter for streaming mode (if different from AdapterType)
	StreamingEndpoint  *EndpointConfig // endpoint for streaming mode (if different from Endpoint)
	SupportedLanguages []string        // explicit list of provider language codes
	Endpoint           *EndpointConfig // nil for local models
	LocalInfo          *LocalModelInfo // nil for cloud models
	DocsURL            string          // URL to provider's language support documentation
}

// EndpointConfig holds HTTP/WebSocket endpoint configuration
type EndpointConfig struct {
	BaseURL string // e.g., "https://api.openai.com" or "wss://api.deepgram.com"
	Path    string // e.g., "/v1/audio/transcriptions"
}

// LocalModelInfo holds metadata for downloadable local models
type LocalModelInfo struct {
	Filename    string // e.g., "ggml-base.en.bin"
	Size        string // human readable size (e.g., "142MB")
	DownloadURL string // full URL to download from
}

// NeedsDownload returns true if this is a local model that requires downloading
func (m *Model) NeedsDownload() bool {
	return m.LocalInfo != nil
}

// IsStreaming returns true if this model supports streaming
func (m *Model) IsStreaming() bool {
	return m.SupportsStreaming
}

// SupportsBothModes returns true if this model supports both batch and streaming
func (m *Model) SupportsBothModes() bool {
	return m.SupportsBatch && m.SupportsStreaming
}

// SupportsLanguage returns true if the model supports the given language code.
// Auto-detect (empty string) is always supported.
func (m *Model) SupportsLanguage(code string) bool {
	if code == "" {
		return true // auto always supported
	}
	for _, supported := range m.SupportedLanguages {
		if supported == code {
			return true
		}
	}
	return false
}
