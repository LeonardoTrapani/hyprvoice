package config

import (
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/history"
)

// DefaultConfig returns the initial configuration used for onboarding.
func DefaultConfig() *Config {
	return &Config{
		Recording: RecordingConfig{
			SampleRate:        16000,
			Channels:          1,
			Format:            "s16",
			BufferSize:        8192,
			Device:            "",
			ChannelBufferSize: 30,
			Timeout:           5 * time.Minute,
		},
		Transcription: TranscriptionConfig{
			Language:  "",
			Streaming: false,
			Threads:   0,
		},
		Injection: InjectionConfig{
			Backends:         []string{"ydotool", "wtype", "clipboard"},
			YdotoolTimeout:   5 * time.Second,
			WtypeTimeout:     5 * time.Second,
			ClipboardTimeout: 3 * time.Second,
		},
		Notifications: NotificationsConfig{
			Enabled: false,
			Type:    "",
		},
		Providers: make(map[string]ProviderConfig),
		Keywords:  nil,
		LLM: LLMConfig{
			Enabled: false,
		},
		History: HistoryConfig{
			// On by default: the whole point is to still have the text when
			// something went wrong, which is not a moment you can plan for.
			Enabled:    true,
			MaxEntries: history.DefaultMaxEntries,
		},
	}
}
