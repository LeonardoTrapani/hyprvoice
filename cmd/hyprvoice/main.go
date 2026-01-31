package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/leonardotrapani/hyprvoice/internal/bus"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/daemon"
	"github.com/leonardotrapani/hyprvoice/internal/tui"
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
	var onboarding bool

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Interactive configuration setup",
		Long: `Interactive configuration wizard for hyprvoice.
This will guide you through setting up:
- Provider API keys (OpenAI, Groq, Mistral, ElevenLabs)
- Transcription settings
- LLM post-processing
- Text injection and notification preferences`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigure(onboarding)
		},
	}

	cmd.Flags().BoolVar(&onboarding, "onboarding", false, "Run the guided onboarding wizard")

	return cmd
}

func runConfigure(onboarding bool) error {
	// Load existing config or create default
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Run TUI wizard
	result, err := tui.Run(cfg, onboarding)
	if err != nil {
		return fmt.Errorf("configuration wizard error: %w", err)
	}

	if result.Cancelled {
		fmt.Println("Configuration cancelled.")
		return nil
	}

	// Validate configuration
	if err := result.Config.Validate(); err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
		return err
	}

	// Save configuration
	if err := saveConfig(result.Config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Println("Configuration saved successfully!")
	fmt.Println()

	// Show next steps
	showNextSteps(result.Config)

	return nil
}

func showNextSteps(cfg *config.Config) {
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

	fmt.Println("Next Steps:")
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
	fmt.Printf("%d. Test voice input: hyprvoice toggle\n", step)
	fmt.Println()

	configPath, _ := config.GetConfigPath()
	fmt.Printf("Config file location: %s\n", configPath)
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

	var sb strings.Builder

	// Header
	sb.WriteString(`# Hyprvoice Configuration
# Generated by hyprvoice configure
# Changes are applied immediately without daemon restart.

`)

	// Keywords (must be before any table definitions)
	if len(cfg.Keywords) > 0 {
		sb.WriteString("# Keywords help transcription and LLM spell names/terms correctly\n")
		sb.WriteString("keywords = [")
		for i, kw := range cfg.Keywords {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%q", kw))
		}
		sb.WriteString("]\n\n")
	}

	// Providers section
	if len(cfg.Providers) > 0 {
		sb.WriteString("# API Keys for providers\n")
		for name, pc := range cfg.Providers {
			sb.WriteString(fmt.Sprintf("[providers.%s]\n", name))
			sb.WriteString(fmt.Sprintf("  api_key = %q\n", pc.APIKey))
			sb.WriteString("\n")
		}
	}

	// Recording
	sb.WriteString(`# Audio Recording Configuration
[recording]
`)
	sb.WriteString(fmt.Sprintf("  sample_rate = %d\n", cfg.Recording.SampleRate))
	sb.WriteString(fmt.Sprintf("  channels = %d\n", cfg.Recording.Channels))
	sb.WriteString(fmt.Sprintf("  format = %q\n", cfg.Recording.Format))
	sb.WriteString(fmt.Sprintf("  buffer_size = %d\n", cfg.Recording.BufferSize))
	sb.WriteString(fmt.Sprintf("  device = %q\n", cfg.Recording.Device))
	sb.WriteString(fmt.Sprintf("  channel_buffer_size = %d\n", cfg.Recording.ChannelBufferSize))
	sb.WriteString(fmt.Sprintf("  timeout = %q\n", cfg.Recording.Timeout.String()))
	sb.WriteString("\n")

	// Transcription
	sb.WriteString(`# Speech Transcription Configuration
[transcription]
`)
	sb.WriteString(fmt.Sprintf("  provider = %q\n", cfg.Transcription.Provider))
	sb.WriteString(fmt.Sprintf("  language = %q\n", cfg.Transcription.Language))
	sb.WriteString(fmt.Sprintf("  model = %q\n", cfg.Transcription.Model))
	sb.WriteString("\n")

	// LLM
	sb.WriteString(`# LLM Post-Processing Configuration
[llm]
`)
	sb.WriteString(fmt.Sprintf("  enabled = %v\n", cfg.LLM.Enabled))
	if cfg.LLM.Provider != "" {
		sb.WriteString(fmt.Sprintf("  provider = %q\n", cfg.LLM.Provider))
	}
	if cfg.LLM.Model != "" {
		sb.WriteString(fmt.Sprintf("  model = %q\n", cfg.LLM.Model))
	}
	sb.WriteString("\n")

	sb.WriteString("  [llm.post_processing]\n")
	sb.WriteString(fmt.Sprintf("    remove_stutters = %v\n", cfg.LLM.PostProcessing.RemoveStutters))
	sb.WriteString(fmt.Sprintf("    add_punctuation = %v\n", cfg.LLM.PostProcessing.AddPunctuation))
	sb.WriteString(fmt.Sprintf("    fix_grammar = %v\n", cfg.LLM.PostProcessing.FixGrammar))
	sb.WriteString(fmt.Sprintf("    remove_filler_words = %v\n", cfg.LLM.PostProcessing.RemoveFillerWords))
	sb.WriteString("\n")

	sb.WriteString("  [llm.custom_prompt]\n")
	sb.WriteString(fmt.Sprintf("    enabled = %v\n", cfg.LLM.CustomPrompt.Enabled))
	if cfg.LLM.CustomPrompt.Prompt != "" {
		sb.WriteString(fmt.Sprintf("    prompt = %q\n", cfg.LLM.CustomPrompt.Prompt))
	}
	sb.WriteString("\n")

	// Injection
	sb.WriteString(`# Text Injection Configuration
[injection]
`)
	sb.WriteString("  backends = [")
	for i, b := range cfg.Injection.Backends {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%q", b))
	}
	sb.WriteString("]\n")
	sb.WriteString(fmt.Sprintf("  ydotool_timeout = %q\n", cfg.Injection.YdotoolTimeout.String()))
	sb.WriteString(fmt.Sprintf("  wtype_timeout = %q\n", cfg.Injection.WtypeTimeout.String()))
	sb.WriteString(fmt.Sprintf("  clipboard_timeout = %q\n", cfg.Injection.ClipboardTimeout.String()))
	sb.WriteString("\n")

	// Notifications
	sb.WriteString(`# Desktop Notification Configuration
[notifications]
`)
	sb.WriteString(fmt.Sprintf("  enabled = %v\n", cfg.Notifications.Enabled))
	sb.WriteString(fmt.Sprintf("  type = %q\n", cfg.Notifications.Type))

	// Write custom messages if any
	msgs := cfg.Notifications.Messages
	if hasCustomMessages(msgs) {
		sb.WriteString("\n  [notifications.messages]\n")
		if msgs.RecordingStarted.Title != "" || msgs.RecordingStarted.Body != "" {
			sb.WriteString("    [notifications.messages.recording_started]\n")
			sb.WriteString(fmt.Sprintf("      title = %q\n", msgs.RecordingStarted.Title))
			sb.WriteString(fmt.Sprintf("      body = %q\n", msgs.RecordingStarted.Body))
		}
		if msgs.Transcribing.Title != "" || msgs.Transcribing.Body != "" {
			sb.WriteString("    [notifications.messages.transcribing]\n")
			sb.WriteString(fmt.Sprintf("      title = %q\n", msgs.Transcribing.Title))
			sb.WriteString(fmt.Sprintf("      body = %q\n", msgs.Transcribing.Body))
		}
		if msgs.LLMProcessing.Title != "" || msgs.LLMProcessing.Body != "" {
			sb.WriteString("    [notifications.messages.llm_processing]\n")
			sb.WriteString(fmt.Sprintf("      title = %q\n", msgs.LLMProcessing.Title))
			sb.WriteString(fmt.Sprintf("      body = %q\n", msgs.LLMProcessing.Body))
		}
		if msgs.ConfigReloaded.Title != "" || msgs.ConfigReloaded.Body != "" {
			sb.WriteString("    [notifications.messages.config_reloaded]\n")
			sb.WriteString(fmt.Sprintf("      title = %q\n", msgs.ConfigReloaded.Title))
			sb.WriteString(fmt.Sprintf("      body = %q\n", msgs.ConfigReloaded.Body))
		}
		if msgs.OperationCancelled.Title != "" || msgs.OperationCancelled.Body != "" {
			sb.WriteString("    [notifications.messages.operation_cancelled]\n")
			sb.WriteString(fmt.Sprintf("      title = %q\n", msgs.OperationCancelled.Title))
			sb.WriteString(fmt.Sprintf("      body = %q\n", msgs.OperationCancelled.Body))
		}
		if msgs.RecordingAborted.Body != "" {
			sb.WriteString("    [notifications.messages.recording_aborted]\n")
			sb.WriteString(fmt.Sprintf("      body = %q\n", msgs.RecordingAborted.Body))
		}
		if msgs.InjectionAborted.Body != "" {
			sb.WriteString("    [notifications.messages.injection_aborted]\n")
			sb.WriteString(fmt.Sprintf("      body = %q\n", msgs.InjectionAborted.Body))
		}
	}

	if _, err := file.WriteString(sb.String()); err != nil {
		return fmt.Errorf("failed to write config content: %w", err)
	}

	return nil
}

func hasCustomMessages(msgs config.MessagesConfig) bool {
	return msgs.RecordingStarted.Title != "" || msgs.RecordingStarted.Body != "" ||
		msgs.Transcribing.Title != "" || msgs.Transcribing.Body != "" ||
		msgs.LLMProcessing.Title != "" || msgs.LLMProcessing.Body != "" ||
		msgs.ConfigReloaded.Title != "" || msgs.ConfigReloaded.Body != "" ||
		msgs.OperationCancelled.Title != "" || msgs.OperationCancelled.Body != "" ||
		msgs.RecordingAborted.Body != "" ||
		msgs.InjectionAborted.Body != ""
}
