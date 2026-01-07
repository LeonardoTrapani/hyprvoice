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
	"github.com/leonardotrapani/hyprvoice/internal/notify"
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
		cancelCmd(),
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

func cancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel",
		Short: "Cancel current operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := bus.SendCommand('c')
			if err != nil {
				return fmt.Errorf("failed to cancel operation: %w", err)
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
- Transcription provider (OpenAI, Groq, or Mistral)
- API keys and model selection
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

	// Provider selection
	for {
		fmt.Println("Select transcription provider:")
		fmt.Println("  1. openai               - OpenAI Whisper API (cloud-based)")
		fmt.Println("  2. groq-transcription   - Groq Whisper API (fast transcription)")
		fmt.Println("  3. groq-translation     - Groq Whisper API (translate to English)")
		fmt.Println("  4. mistral-transcription - Mistral Voxtral API (excellent for European languages)")
		fmt.Printf("Provider [1-4] (current: %s): ", cfg.Transcription.Provider)
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			break // keep current
		}
		switch input {
		case "1":
			cfg.Transcription.Provider = "openai"
		case "2":
			cfg.Transcription.Provider = "groq-transcription"
		case "3":
			cfg.Transcription.Provider = "groq-translation"
		case "4":
			cfg.Transcription.Provider = "mistral-transcription"
		case "openai", "groq-transcription", "groq-translation", "mistral-transcription":
			cfg.Transcription.Provider = input
		default:
			fmt.Println("❌ Error: invalid provider. Please enter 1-4 or provider name.")
			fmt.Println()
			continue
		}
		break
	}

	// Model selection based on provider
	switch cfg.Transcription.Provider {
	case "openai":
		fmt.Println("\nOpenAI Model:")
		fmt.Printf("Model (current: %s): ", cfg.Transcription.Model)
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input != "" {
				cfg.Transcription.Model = input
			} else if cfg.Transcription.Model == "" {
				cfg.Transcription.Model = "whisper-1"
			}
		}
	case "groq-transcription":
		for {
			fmt.Println("\nGroq Transcription Model:")
			fmt.Println("  1. whisper-large-v3       - Standard model")
			fmt.Println("  2. whisper-large-v3-turbo - Faster model")
			fmt.Printf("Model [1-2] (current: %s): ", cfg.Transcription.Model)
			if !scanner.Scan() {
				break
			}
			input := strings.TrimSpace(scanner.Text())
			switch input {
			case "1":
				cfg.Transcription.Model = "whisper-large-v3"
			case "2":
				cfg.Transcription.Model = "whisper-large-v3-turbo"
			case "whisper-large-v3", "whisper-large-v3-turbo":
				cfg.Transcription.Model = input
			case "":
				if cfg.Transcription.Model == "" {
					cfg.Transcription.Model = "whisper-large-v3-turbo"
				}
			default:
				fmt.Println("❌ Error: invalid model. Please enter 1, 2 or model name.")
				continue
			}
			break
		}
	case "groq-translation":
		for {
			fmt.Println("\nGroq Translation Model:")
			fmt.Println("  Note: Translation only supports whisper-large-v3 (turbo not available)")
			fmt.Printf("Model (current: %s, press Enter for whisper-large-v3): ", cfg.Transcription.Model)
			if !scanner.Scan() {
				break
			}
			input := strings.TrimSpace(scanner.Text())
			if input == "" || input == "whisper-large-v3" || input == "1" {
				cfg.Transcription.Model = "whisper-large-v3"
				break
			}
			fmt.Println("❌ Error: only whisper-large-v3 is supported for translation.")
		}
	case "mistral-transcription":
		for {
			fmt.Println("\nMistral Voxtral Model:")
			fmt.Println("  1. voxtral-mini-latest - Recommended (latest version)")
			fmt.Println("  2. voxtral-mini-2507   - Pinned version")
			fmt.Printf("Model [1-2] (current: %s): ", cfg.Transcription.Model)
			if !scanner.Scan() {
				break
			}
			input := strings.TrimSpace(scanner.Text())
			switch input {
			case "1":
				cfg.Transcription.Model = "voxtral-mini-latest"
			case "2":
				cfg.Transcription.Model = "voxtral-mini-2507"
			case "voxtral-mini-latest", "voxtral-mini-2507":
				cfg.Transcription.Model = input
			case "":
				if cfg.Transcription.Model == "" || !strings.HasPrefix(cfg.Transcription.Model, "voxtral") {
					cfg.Transcription.Model = "voxtral-mini-latest"
				}
			default:
				fmt.Println("❌ Error: invalid model. Please enter 1, 2 or model name.")
				continue
			}
			break
		}
	}

	// API Key (provider-aware)
	var envVarName string
	switch cfg.Transcription.Provider {
	case "openai":
		envVarName = "OPENAI_API_KEY"
	case "mistral-transcription":
		envVarName = "MISTRAL_API_KEY"
	default:
		envVarName = "GROQ_API_KEY"
	}
	fmt.Printf("\nAPI Key (current: %s, leave empty to use %s env var): ", maskAPIKey(cfg.Transcription.APIKey), envVarName)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			cfg.Transcription.APIKey = input
		}
	}

	// Language
	if cfg.Transcription.Provider == "groq-translation" {
		fmt.Printf("\nSource language hint (empty for auto-detect, current: %s): ", cfg.Transcription.Language)
		fmt.Println("\n  Note: Translation always outputs English. Language hints at source audio language.")
	} else {
		fmt.Printf("\nLanguage (empty for auto-detect, current: %s): ", cfg.Transcription.Language)
	}
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		cfg.Transcription.Language = input
	}

	fmt.Println()

	// Configure injection
	for {
		fmt.Println("⌨️  Text Injection Configuration")
		fmt.Println("--------------------------------")
		fmt.Println("Backends are tried in order until one succeeds (fallback chain):")
		fmt.Println("  - ydotool:   Best for Chromium/Electron apps (requires ydotoold daemon for ydotool v1.0.0+)")
		fmt.Println("  - wtype:     Native Wayland typing (may fail on some Chromium apps)")
		fmt.Println("  - clipboard: Copies to clipboard only (most reliable, needs manual paste)")
		fmt.Println()
		fmt.Println("Recommended: ydotool,wtype,clipboard (full fallback chain)")
		fmt.Println()
		fmt.Printf("Backends (comma-separated) (current: %s): ", strings.Join(cfg.Injection.Backends, ","))
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			break // keep current
		}
		backends := strings.Split(input, ",")
		validBackends := make([]string, 0)
		invalidBackends := make([]string, 0)
		for _, b := range backends {
			b = strings.TrimSpace(b)
			if b == "ydotool" || b == "wtype" || b == "clipboard" {
				validBackends = append(validBackends, b)
			} else if b != "" {
				invalidBackends = append(invalidBackends, b)
			}
		}
		if len(invalidBackends) > 0 {
			fmt.Printf("❌ Error: invalid backend(s): %s. Valid: ydotool, wtype, clipboard.\n", strings.Join(invalidBackends, ", "))
			fmt.Println()
			continue
		}
		if len(validBackends) == 0 {
			fmt.Println("❌ Error: at least one backend required.")
			fmt.Println()
			continue
		}
		cfg.Injection.Backends = validBackends
		break
	}

	// Check if ydotool is selected and warn about daemon requirement
	for _, b := range cfg.Injection.Backends {
		if b == "ydotool" {
			fmt.Println()
			fmt.Println("⚠️  ydotool requires the ydotoold daemon to be running! make sure it works")
			fmt.Println()
			break
		}
	}

	fmt.Println()

	// Configure notifications
	for {
		fmt.Println("🔔 Notification Configuration")
		fmt.Println("-----------------------------")
		fmt.Printf("Enable notifications [y/n] (current: %v): ", cfg.Notifications.Enabled)
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(strings.ToLower(scanner.Text()))
		switch input {
		case "y", "yes":
			cfg.Notifications.Enabled = true
		case "n", "no":
			cfg.Notifications.Enabled = false
		case "":
			// keep current
		default:
			fmt.Println("❌ Error: please enter y or n.")
			fmt.Println()
			continue
		}
		break
	}

	// Ask if user wants to customize notification messages
	fmt.Print("Customize notification messages? [y/n] (default: n): ")
	if scanner.Scan() {
		input := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if input == "y" || input == "yes" {
			fmt.Println()
			// Get resolved values (user config merged with defaults)
			msgs := cfg.Notifications.Messages.Resolve()

			// Recording Started
			fmt.Println("  Recording Started notification:")
			fmt.Printf("    Title (current: %s): ", msgs[notify.MsgRecordingStarted].Title)
			if scanner.Scan() {
				if t := strings.TrimSpace(scanner.Text()); t != "" {
					cfg.Notifications.Messages.RecordingStarted.Title = t
				}
			}
			fmt.Printf("    Body (current: %s): ", msgs[notify.MsgRecordingStarted].Body)
			if scanner.Scan() {
				if b := strings.TrimSpace(scanner.Text()); b != "" {
					cfg.Notifications.Messages.RecordingStarted.Body = b
				}
			}
			fmt.Println()

			// Transcribing
			fmt.Println("  Transcribing notification:")
			fmt.Printf("    Title (current: %s): ", msgs[notify.MsgTranscribing].Title)
			if scanner.Scan() {
				if t := strings.TrimSpace(scanner.Text()); t != "" {
					cfg.Notifications.Messages.Transcribing.Title = t
				}
			}
			fmt.Printf("    Body (current: %s): ", msgs[notify.MsgTranscribing].Body)
			if scanner.Scan() {
				if b := strings.TrimSpace(scanner.Text()); b != "" {
					cfg.Notifications.Messages.Transcribing.Body = b
				}
			}
			fmt.Println()

			// Config Reloaded
			fmt.Println("  Config Reloaded notification:")
			fmt.Printf("    Title (current: %s): ", msgs[notify.MsgConfigReloaded].Title)
			if scanner.Scan() {
				if t := strings.TrimSpace(scanner.Text()); t != "" {
					cfg.Notifications.Messages.ConfigReloaded.Title = t
				}
			}
			fmt.Printf("    Body (current: %s): ", msgs[notify.MsgConfigReloaded].Body)
			if scanner.Scan() {
				if b := strings.TrimSpace(scanner.Text()); b != "" {
					cfg.Notifications.Messages.ConfigReloaded.Body = b
				}
			}
			fmt.Println()

			// Operation Cancelled
			fmt.Println("  Operation Cancelled notification:")
			fmt.Printf("    Title (current: %s): ", msgs[notify.MsgOperationCancelled].Title)
			if scanner.Scan() {
				if t := strings.TrimSpace(scanner.Text()); t != "" {
					cfg.Notifications.Messages.OperationCancelled.Title = t
				}
			}
			fmt.Printf("    Body (current: %s): ", msgs[notify.MsgOperationCancelled].Body)
			if scanner.Scan() {
				if b := strings.TrimSpace(scanner.Text()); b != "" {
					cfg.Notifications.Messages.OperationCancelled.Body = b
				}
			}
			fmt.Println()

			// Recording Aborted (body only)
			fmt.Println("  Recording Aborted notification:")
			fmt.Printf("    Body (current: %s): ", msgs[notify.MsgRecordingAborted].Body)
			if scanner.Scan() {
				if b := strings.TrimSpace(scanner.Text()); b != "" {
					cfg.Notifications.Messages.RecordingAborted.Body = b
				}
			}
			fmt.Println()

			// Injection Aborted (body only)
			fmt.Println("  Injection Aborted notification:")
			fmt.Printf("    Body (current: %s): ", msgs[notify.MsgInjectionAborted].Body)
			if scanner.Scan() {
				if b := strings.TrimSpace(scanner.Text()); b != "" {
					cfg.Notifications.Messages.InjectionAborted.Body = b
				}
			}
		}
	}

	fmt.Println()

	// Configure recording timeout
	for {
		fmt.Println("⏱️  Recording Configuration")
		fmt.Println("---------------------------")
		fmt.Printf("Recording timeout in minutes (current: %.0f): ", cfg.Recording.Timeout.Minutes())
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			break // keep current
		}
		minutes, err := strconv.Atoi(input)
		if err != nil || minutes <= 0 {
			fmt.Println("❌ Error: please enter a positive number.")
			fmt.Println()
			continue
		}
		cfg.Recording.Timeout = time.Duration(minutes) * time.Minute
		break
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

	// Check if ydotool is in backends
	hasYdotool := false
	for _, b := range cfg.Injection.Backends {
		if b == "ydotool" {
			hasYdotool = true
			break
		}
	}

	// Show next steps
	fmt.Println("🚀 Next Steps:")
	step := 1
	if hasYdotool {
		fmt.Printf("%d. Ensure ydotoold is running\n", step)
		step++
	}
	if !serviceRunning {
		fmt.Printf("%d. Start the service: systemctl --user start hyprvoice.service\n", step)
	} else {
		fmt.Printf("%d. Restart the service to apply changes: systemctl --user restart hyprvoice.service\n", step)
	}
	step++
	fmt.Printf("%d. Test voice input: hyprvoice toggle (or use keybind you configured in hyprland config)\n", step)
	fmt.Println()

	configPath, _ := config.GetConfigPath()
	fmt.Printf("📁 Config file location: %s\n", configPath)

	return nil
}

func formatBackends(backends []string) string {
	quoted := make([]string, len(backends))
	for i, b := range backends {
		quoted[i] = fmt.Sprintf(`"%s"`, b)
	}
	return strings.Join(quoted, ", ")
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
  provider = "%s"          # Transcription service: "openai", "groq-transcription", "groq-translation", or "mistral-transcription"
  api_key = "%s"                 # API key (or set OPENAI_API_KEY/GROQ_API_KEY/MISTRAL_API_KEY environment variable)
  language = "%s"                # Language code (empty for auto-detect, "en", "it", "es", "fr", etc.)
  model = "%s"          # Model: OpenAI="whisper-1", Groq="whisper-large-v3", Mistral="voxtral-mini-latest"

# Text Injection Configuration
[injection]
  backends = [%s]  # Ordered fallback chain (tries each until one succeeds)
  ydotool_timeout = "%s"       # Timeout for ydotool commands
  wtype_timeout = "%s"         # Timeout for wtype commands
  clipboard_timeout = "%s"     # Timeout for clipboard operations

# Backend explanations:
# - "ydotool": Uses ydotool (requires ydotoold daemon running for ydotool v1.0.0+). Most compatible with Chromium/Electron apps.
# - "wtype": Uses wtype for Wayland. May have issues with some Chromium-based apps.
# - "clipboard": Copies text to clipboard only (most reliable, but requires manual paste).
#
# The backends are tried in order. First successful one wins.
#
# Provider explanations:
# - "openai": OpenAI Whisper API (cloud-based, requires OPENAI_API_KEY)
# - "groq-transcription": Groq Whisper API for transcription (fast, requires GROQ_API_KEY)
#     Models: whisper-large-v3 or whisper-large-v3-turbo
# - "groq-translation": Groq Whisper API for translation to English (always outputs English text)
#     Models: whisper-large-v3 only (turbo not supported for translation)
# - "mistral-transcription": Mistral Voxtral API (excellent for European languages, requires MISTRAL_API_KEY)
#     Models: voxtral-mini-latest or voxtral-mini-2507
#
# Language codes: Use empty string ("") for automatic detection, or specific codes like:
# "en" (English), "it" (Italian), "es" (Spanish), "fr" (French), "de" (German), etc.
# For groq-translation, the language field hints at the source audio language for better accuracy.

# Desktop Notification Configuration
[notifications]
  enabled = %v               # Enable desktop notifications
  type = "%s"             # Notification type ("desktop", "log", "none")
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
		formatBackends(cfg.Injection.Backends),
		cfg.Injection.YdotoolTimeout,
		cfg.Injection.WtypeTimeout,
		cfg.Injection.ClipboardTimeout,
		cfg.Notifications.Enabled,
		cfg.Notifications.Type,
	)

	if _, err := file.WriteString(configContent); err != nil {
		return fmt.Errorf("failed to write config content: %w", err)
	}

	// Write notification messages if any are configured
	msgs := cfg.Notifications.Messages
	if hasCustomMessages(msgs) {
		messagesContent := "\n  [notifications.messages]\n"
		if msgs.RecordingStarted.Title != "" || msgs.RecordingStarted.Body != "" {
			messagesContent += fmt.Sprintf("    [notifications.messages.recording_started]\n      title = %q\n      body = %q\n",
				msgs.RecordingStarted.Title, msgs.RecordingStarted.Body)
		}
		if msgs.Transcribing.Title != "" || msgs.Transcribing.Body != "" {
			messagesContent += fmt.Sprintf("    [notifications.messages.transcribing]\n      title = %q\n      body = %q\n",
				msgs.Transcribing.Title, msgs.Transcribing.Body)
		}
		if msgs.ConfigReloaded.Title != "" || msgs.ConfigReloaded.Body != "" {
			messagesContent += fmt.Sprintf("    [notifications.messages.config_reloaded]\n      title = %q\n      body = %q\n",
				msgs.ConfigReloaded.Title, msgs.ConfigReloaded.Body)
		}
		if msgs.OperationCancelled.Title != "" || msgs.OperationCancelled.Body != "" {
			messagesContent += fmt.Sprintf("    [notifications.messages.operation_cancelled]\n      title = %q\n      body = %q\n",
				msgs.OperationCancelled.Title, msgs.OperationCancelled.Body)
		}
		if msgs.RecordingAborted.Body != "" {
			messagesContent += fmt.Sprintf("    [notifications.messages.recording_aborted]\n      body = %q\n",
				msgs.RecordingAborted.Body)
		}
		if msgs.InjectionAborted.Body != "" {
			messagesContent += fmt.Sprintf("    [notifications.messages.injection_aborted]\n      body = %q\n",
				msgs.InjectionAborted.Body)
		}
		if _, err := file.WriteString(messagesContent); err != nil {
			return fmt.Errorf("failed to write messages config: %w", err)
		}
	}

	return nil
}

func hasCustomMessages(msgs config.MessagesConfig) bool {
	return msgs.RecordingStarted.Title != "" || msgs.RecordingStarted.Body != "" ||
		msgs.Transcribing.Title != "" || msgs.Transcribing.Body != "" ||
		msgs.ConfigReloaded.Title != "" || msgs.ConfigReloaded.Body != "" ||
		msgs.OperationCancelled.Title != "" || msgs.OperationCancelled.Body != "" ||
		msgs.RecordingAborted.Body != "" ||
		msgs.InjectionAborted.Body != ""
}
