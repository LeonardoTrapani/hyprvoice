package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/bus"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/daemon"
	"github.com/spf13/cobra"
)

func main() {
	_ = rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "hyprvoice",
	Short: "Voice-powered typing for Wayland/Hyprland",
}

func init() {
	rootCmd.AddCommand(
		serveCmd(),
		toggleCmd(),
		statusCmd(),
		versionCmd(),
		stopCmd(),
		configureCmd(),
	)
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := daemon.New()
			if err != nil {
				return fmt.Errorf("failed to create daemon: %w", err)
			}
			return d.Run()
		},
	}
}

func toggleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "toggle",
		Short: "Toggle recording on/off",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := bus.SendCommand('t')
			if err != nil {
				return fmt.Errorf("failed to toggle recording: %w", err)
			}
			fmt.Print(resp)
			return nil
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Get current recording status",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := bus.SendCommand('s')
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}
			fmt.Print(resp)
			return nil
		},
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Get protocol version",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := bus.SendCommand('v')
			if err != nil {
				return fmt.Errorf("failed to get version: %w", err)
			}
			fmt.Print(resp)
			return nil
		},
	}
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := bus.SendCommand('q')
			if err != nil {
				return fmt.Errorf("failed to stop daemon: %w", err)
			}
			fmt.Print(resp)
			return nil
		},
	}
}

func configureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "configure",
		Short: "Interactive configuration setup",
		Long: `Interactive configuration wizard for hyprvoice.
This will guide you through setting up:
- OpenAI API key for transcription
- Audio and text injection preferences
- Notification settings`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInteractiveConfig()
		},
	}
}

func runInteractiveConfig() error {
	fmt.Println("🎤 Hyprvoice Configuration Wizard")
	fmt.Println("==================================")
	fmt.Println()

	// Load existing config or create default
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	scanner := bufio.NewScanner(os.Stdin)

	// Configure transcription
	fmt.Println("📝 Transcription Configuration")
	fmt.Println("------------------------------")

	// OpenAI API Key
	fmt.Printf("OpenAI API Key (current: %s, leave empty to use OPENAI_API_KEY env var): ", maskAPIKey(cfg.Transcription.APIKey))
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			cfg.Transcription.APIKey = input
		}
	}

	// Language
	fmt.Printf("Language (empty for auto-detect, current: %s): ", cfg.Transcription.Language)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		cfg.Transcription.Language = input
	}

	fmt.Println()

	// Configure injection
	fmt.Println("⌨️  Text Injection Configuration")
	fmt.Println("--------------------------------")
	fmt.Printf("Injection mode [clipboard/type/fallback] (current: %s): ", cfg.Injection.Mode)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" && (input == "clipboard" || input == "type" || input == "fallback") {
			cfg.Injection.Mode = input
		}
	}

	fmt.Printf("Restore clipboard after injection [y/n] (current: %v): ", cfg.Injection.RestoreClipboard)
	if scanner.Scan() {
		switch strings.TrimSpace(strings.ToLower(scanner.Text())) {
		case "y", "yes":
			cfg.Injection.RestoreClipboard = true
		case "n", "no":
			cfg.Injection.RestoreClipboard = false
		}
	}

	fmt.Println()

	// Configure notifications
	fmt.Println("🔔 Notification Configuration")
	fmt.Println("-----------------------------")
	fmt.Printf("Enable notifications [y/n] (current: %v): ", cfg.Notifications.Enabled)
	if scanner.Scan() {
		switch strings.TrimSpace(strings.ToLower(scanner.Text())) {
		case "y", "yes":
			cfg.Notifications.Enabled = true
		case "n", "no":
			cfg.Notifications.Enabled = false
		}
	}

	fmt.Println()

	// Configure recording timeout
	fmt.Println("⏱️  Recording Configuration")
	fmt.Println("---------------------------")
	fmt.Printf("Recording timeout in minutes (current: %.0f): ", cfg.Recording.Timeout.Minutes())
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			if minutes, err := strconv.Atoi(input); err == nil && minutes > 0 {
				cfg.Recording.Timeout = time.Duration(minutes) * time.Minute
			}
		}
	}

	fmt.Println()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Printf("❌ Configuration validation failed: %v\n", err)
		fmt.Println("Please check your inputs and try again.")
		return err
	}

	// Save configuration
	fmt.Println("💾 Saving configuration...")
	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("✅ Configuration saved successfully!")
	fmt.Println()

	// Check if service is running
	serviceRunning := false
	if _, err := exec.Command("systemctl", "--user", "is-active", "--quiet", "hyprvoice.service").CombinedOutput(); err == nil {
		serviceRunning = true
	}

	// Show next steps
	fmt.Println("🚀 Next Steps:")
	if !serviceRunning {
		fmt.Println("1. Start the service: systemctl --user start hyprvoice.service")
		fmt.Println("2. Test voice input: hyprvoice toggle")
	} else {
		fmt.Println("1. Restart the service to apply changes: systemctl --user restart hyprvoice.service")
		fmt.Println("2. Test voice input: hyprvoice toggle")
	}
	fmt.Println()

	configPath, _ := config.GetConfigPath()
	fmt.Printf("📁 Config file location: %s\n", configPath)

	return nil
}

func maskAPIKey(key string) string {
	if key == "" {
		return "<not set>"
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func saveConfig(cfg *config.Config) error {
	configPath, err := config.GetConfigPath()
	if err != nil {
		return err
	}

	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	configContent := fmt.Sprintf(`# Hyprvoice Configuration
# This file is automatically generated with defaults.
# Edit values as needed - changes are applied immediately without daemon restart.

# Audio Recording Configuration
[recording]
  sample_rate = %d          # Audio sample rate in Hz (16000 recommended for speech)
  channels = %d                 # Number of audio channels (1 = mono, 2 = stereo)
  format = "%s"               # Audio format (s16 = 16-bit signed integers)
  buffer_size = %d           # Internal buffer size in bytes (larger = less CPU, more latency)
  device = "%s"                  # PipeWire audio device (empty = use default microphone)
  channel_buffer_size = %d     # Audio frame buffer size (frames to buffer)
  timeout = "%s"               # Maximum recording duration (e.g., "30s", "2m", "5m")

# Speech Transcription Configuration  
[transcription]
  provider = "%s"          # Transcription service ("openai" only currently supported)
  api_key = "%s"                 # OpenAI API key (or set OPENAI_API_KEY environment variable)
  language = "%s"                # Language code (empty for auto-detect, "en", "it", "es", "fr", etc.)
  model = "%s"          # OpenAI model name ("whisper-1" recommended)

# Text Injection Configuration
[injection]
  mode = "%s"            # Injection method ("clipboard", "type", "fallback")
  restore_clipboard = %v     # Restore original clipboard after injection
  wtype_timeout = "%s"         # Timeout for direct typing via wtype
  clipboard_timeout = "%s"     # Timeout for clipboard operations

# Desktop Notification Configuration
[notifications]
  enabled = %v               # Enable desktop notifications
  type = "%s"             # Notification type ("desktop", "log", "none")

# Mode explanations:
# - "clipboard": Copy text to clipboard only
# - "type": Direct typing via wtype only  
# - "fallback": Try typing first, fallback to clipboard if it fails
#
# Language codes: Use empty string ("") for automatic detection, or specific codes like:
# "en" (English), "it" (Italian), "es" (Spanish), "fr" (French), "de" (German), etc.
`,
		cfg.Recording.SampleRate,
		cfg.Recording.Channels,
		cfg.Recording.Format,
		cfg.Recording.BufferSize,
		cfg.Recording.Device,
		cfg.Recording.ChannelBufferSize,
		cfg.Recording.Timeout,
		cfg.Transcription.Provider,
		cfg.Transcription.APIKey,
		cfg.Transcription.Language,
		cfg.Transcription.Model,
		cfg.Injection.Mode,
		cfg.Injection.RestoreClipboard,
		cfg.Injection.WtypeTimeout,
		cfg.Injection.ClipboardTimeout,
		cfg.Notifications.Enabled,
		cfg.Notifications.Type,
	)

	if _, err := file.WriteString(configContent); err != nil {
		return fmt.Errorf("failed to write config content: %w", err)
	}

	return nil
}
