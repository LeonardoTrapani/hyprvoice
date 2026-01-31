package tui

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/leonardotrapani/hyprvoice/internal/config"
)

// AdvancedSection represents a section in the advanced settings menu
type AdvancedSection string

const (
	AdvancedRecording        AdvancedSection = "recording"
	AdvancedInjectionTimeout AdvancedSection = "injection_timeout"
	AdvancedBack             AdvancedSection = "back"
)

// editAdvanced handles the advanced settings submenu
func editAdvanced(cfg *config.Config) error {
	for {
		options := []huh.Option[AdvancedSection]{
			huh.NewOption(formatAdvancedRecordingLabel(cfg), AdvancedRecording),
			huh.NewOption(formatAdvancedInjectionTimeoutLabel(cfg), AdvancedInjectionTimeout),
			huh.NewOption("Back to Main Menu", AdvancedBack),
		}

		var selected AdvancedSection
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[AdvancedSection]().
					Title("Advanced Settings").
					Description("Configure low-level options").
					Options(options...).
					Value(&selected),
			),
		).WithTheme(getTheme())

		if err := form.Run(); err != nil {
			return err
		}

		switch selected {
		case AdvancedBack:
			return nil
		case AdvancedRecording:
			if err := editRecording(cfg); err != nil {
				continue
			}
		case AdvancedInjectionTimeout:
			if err := editInjectionTimeouts(cfg); err != nil {
				continue
			}
		}
	}
}

func formatAdvancedRecordingLabel(cfg *config.Config) string {
	return fmt.Sprintf("Recording Settings (rate=%d, timeout=%s)", cfg.Recording.SampleRate, cfg.Recording.Timeout)
}

func formatAdvancedInjectionTimeoutLabel(cfg *config.Config) string {
	return fmt.Sprintf("Injection Timeouts (ydotool=%s, wtype=%s, clipboard=%s)",
		cfg.Injection.YdotoolTimeout, cfg.Injection.WtypeTimeout, cfg.Injection.ClipboardTimeout)
}

// editRecording handles the recording settings
func editRecording(cfg *config.Config) error {
	sampleRate := strconv.Itoa(cfg.Recording.SampleRate)
	channels := strconv.Itoa(cfg.Recording.Channels)
	format := cfg.Recording.Format
	bufferSize := strconv.Itoa(cfg.Recording.BufferSize)
	device := cfg.Recording.Device
	channelBufferSize := strconv.Itoa(cfg.Recording.ChannelBufferSize)
	timeout := cfg.Recording.Timeout.String()

	channelOptions := []huh.Option[string]{
		huh.NewOption("1 (Mono) - Recommended", "1"),
		huh.NewOption("2 (Stereo)", "2"),
	}

	formatOptions := []huh.Option[string]{
		huh.NewOption("s16 (16-bit signed) - Recommended", "s16"),
		huh.NewOption("f32 (32-bit float)", "f32"),
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Sample Rate (Hz)").
				Description("Audio sample rate. 16000 is optimal for speech recognition.").
				Placeholder("16000").
				Value(&sampleRate).
				Validate(func(s string) error {
					if _, err := strconv.Atoi(s); err != nil {
						return fmt.Errorf("must be a number")
					}
					return nil
				}),
			huh.NewSelect[string]().
				Title("Channels").
				Description("Number of audio channels").
				Options(channelOptions...).
				Value(&channels),
			huh.NewSelect[string]().
				Title("Audio Format").
				Description("Sample format").
				Options(formatOptions...).
				Value(&format),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Buffer Size (bytes)").
				Description("Internal buffer size. Larger = less CPU, more latency.").
				Placeholder("8192").
				Value(&bufferSize).
				Validate(func(s string) error {
					if _, err := strconv.Atoi(s); err != nil {
						return fmt.Errorf("must be a number")
					}
					return nil
				}),
			huh.NewInput().
				Title("Channel Buffer Size").
				Description("Number of audio frames to buffer.").
				Placeholder("30").
				Value(&channelBufferSize).
				Validate(func(s string) error {
					if _, err := strconv.Atoi(s); err != nil {
						return fmt.Errorf("must be a number")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Device").
				Description("PipeWire device name. Empty = default microphone.").
				Placeholder("(default)").
				Value(&device),
			huh.NewInput().
				Title("Recording Timeout").
				Description("Max recording duration (e.g., '30s', '2m', '5m'). Prevents runaway recordings.").
				Placeholder("5m").
				Value(&timeout).
				Validate(func(s string) error {
					if _, err := time.ParseDuration(s); err != nil {
						return fmt.Errorf("invalid duration format (use '30s', '2m', etc.)")
					}
					return nil
				}),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return err
	}

	cfg.Recording.SampleRate, _ = strconv.Atoi(sampleRate)
	cfg.Recording.Channels, _ = strconv.Atoi(channels)
	cfg.Recording.Format = format
	cfg.Recording.BufferSize, _ = strconv.Atoi(bufferSize)
	cfg.Recording.Device = device
	cfg.Recording.ChannelBufferSize, _ = strconv.Atoi(channelBufferSize)
	cfg.Recording.Timeout, _ = time.ParseDuration(timeout)

	return nil
}

// editInjectionTimeouts handles the injection timeout settings
func editInjectionTimeouts(cfg *config.Config) error {
	ydotoolTimeout := cfg.Injection.YdotoolTimeout.String()
	wtypeTimeout := cfg.Injection.WtypeTimeout.String()
	clipboardTimeout := cfg.Injection.ClipboardTimeout.String()

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("ydotool Timeout").
				Description("Timeout for ydotool commands (e.g., '5s', '10s')").
				Placeholder("5s").
				Value(&ydotoolTimeout).
				Validate(func(s string) error {
					if _, err := time.ParseDuration(s); err != nil {
						return fmt.Errorf("invalid duration format")
					}
					return nil
				}),
			huh.NewInput().
				Title("wtype Timeout").
				Description("Timeout for wtype commands (e.g., '5s', '10s')").
				Placeholder("5s").
				Value(&wtypeTimeout).
				Validate(func(s string) error {
					if _, err := time.ParseDuration(s); err != nil {
						return fmt.Errorf("invalid duration format")
					}
					return nil
				}),
			huh.NewInput().
				Title("Clipboard Timeout").
				Description("Timeout for clipboard operations (e.g., '3s', '5s')").
				Placeholder("3s").
				Value(&clipboardTimeout).
				Validate(func(s string) error {
					if _, err := time.ParseDuration(s); err != nil {
						return fmt.Errorf("invalid duration format")
					}
					return nil
				}),
		),
	).WithTheme(getTheme())

	if err := form.Run(); err != nil {
		return err
	}

	cfg.Injection.YdotoolTimeout, _ = time.ParseDuration(ydotoolTimeout)
	cfg.Injection.WtypeTimeout, _ = time.ParseDuration(wtypeTimeout)
	cfg.Injection.ClipboardTimeout, _ = time.ParseDuration(clipboardTimeout)

	return nil
}
