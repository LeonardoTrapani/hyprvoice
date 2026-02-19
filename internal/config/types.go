package config

import (
	"reflect"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/notify"
)

// GeneralConfig holds global settings that apply across the application
type GeneralConfig struct {
	// reserved for future use
}

type Config struct {
	General       GeneralConfig             `toml:"general"`
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
	Provider  string `toml:"provider"`
	Language  string `toml:"language"`
	Model     string `toml:"model"`
	Streaming bool   `toml:"streaming"` // use streaming mode if model supports it
	Threads   int    `toml:"threads"`   // CPU threads for local transcription (0 = auto: NumCPU-1)
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
	LLMProcessing      MessageConfig `toml:"llm_processing"`
	ConfigReloaded     MessageConfig `toml:"config_reloaded"`
	OperationCancelled MessageConfig `toml:"operation_cancelled"`
	RecordingAborted   MessageConfig `toml:"recording_aborted"`
	InjectionAborted   MessageConfig `toml:"injection_aborted"`
	InjectionComplete  MessageConfig `toml:"injection_complete"`
}

// Resolve merges user config with defaults from MessageDefs
func (m *MessagesConfig) Resolve() map[notify.MessageType]notify.Message {
	result := make(map[notify.MessageType]notify.Message)

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
