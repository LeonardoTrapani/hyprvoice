package config

import "time"

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
	}
}
