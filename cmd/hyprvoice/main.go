package main

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/leonardotrapani/hyprvoice/internal/bus"
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/daemon"
	"github.com/leonardotrapani/hyprvoice/internal/models/whisper"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
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
		modelCmd(),
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
	if err := config.Save(result.Config); err != nil {
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

func modelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Manage transcription models",
	}

	cmd.AddCommand(modelListCmd())
	cmd.AddCommand(modelDownloadCmd())
	cmd.AddCommand(modelRemoveCmd())

	return cmd
}

func modelListCmd() *cobra.Command {
	var providerFilter string
	var typeFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available transcription and LLM models",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelList(providerFilter, typeFilter)
		},
	}

	cmd.Flags().StringVar(&providerFilter, "provider", "", "filter by provider name")
	cmd.Flags().StringVar(&typeFilter, "type", "", "filter by type: transcription, llm")

	return cmd
}

func runModelList(providerFilter, typeFilter string) error {
	// parse type filter
	var filterType *provider.ModelType
	if typeFilter != "" {
		switch strings.ToLower(typeFilter) {
		case "transcription":
			t := provider.Transcription
			filterType = &t
		case "llm":
			t := provider.LLM
			filterType = &t
		default:
			return fmt.Errorf("invalid type: %s (use 'transcription' or 'llm')", typeFilter)
		}
	}

	// get providers to iterate
	providerNames := provider.ListProviders()
	sort.Strings(providerNames)

	// filter by provider if specified
	if providerFilter != "" {
		found := false
		for _, name := range providerNames {
			if name == providerFilter {
				providerNames = []string{name}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown provider: %s", providerFilter)
		}
	}

	for _, providerName := range providerNames {
		p := provider.GetProvider(providerName)
		if p == nil {
			continue
		}

		models := p.Models()
		if filterType != nil {
			models = provider.ModelsOfType(p, *filterType)
		}

		if len(models) == 0 {
			continue
		}

		// print provider header
		fmt.Printf("\n%s:\n", providerName)

		for _, m := range models {
			printModelLine(m)
		}
	}

	fmt.Println()
	return nil
}

func printModelLine(m provider.Model) {
	// build prefix: checkmark for installed local models
	prefix := "  "
	if m.Local {
		if whisper.IsInstalled(m.ID) {
			prefix = "  [x]"
		} else {
			prefix = "  [ ]"
		}
	}

	// build suffix parts
	var parts []string

	// type indicator
	if m.Type == provider.LLM {
		parts = append(parts, "llm")
	}

	// mode capabilities indicator
	if m.SupportsBothModes() {
		parts = append(parts, "batch+streaming")
	} else if m.SupportsStreaming {
		parts = append(parts, "streaming")
	}

	// size for local models
	if m.LocalInfo != nil && m.LocalInfo.Size != "" {
		parts = append(parts, m.LocalInfo.Size)
	}

	// build line
	line := fmt.Sprintf("%s %s", prefix, m.ID)
	if m.Description != "" {
		line += fmt.Sprintf(" - %s", m.Description)
	}
	if len(parts) > 0 {
		line += fmt.Sprintf(" [%s]", strings.Join(parts, ", "))
	}

	fmt.Println(line)
}

func modelDownloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "download <model-name>",
		Short: "Download a local model (e.g. whisper models)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelDownload(cmd.Context(), args[0])
		},
	}
}

func runModelDownload(ctx context.Context, modelName string) error {
	// find the model across all providers
	model, _, err := provider.FindModelByID(modelName)
	if err != nil {
		return fmt.Errorf("unknown model: %s", modelName)
	}

	// check if it needs download (local model)
	if !model.NeedsDownload() {
		fmt.Printf("model '%s' is a cloud model and does not require download\n", modelName)
		return nil
	}

	// check if already installed
	if whisper.IsInstalled(modelName) {
		path := whisper.GetModelPath(modelName)
		fmt.Printf("model '%s' is already installed at %s\n", modelName, path)
		return nil
	}

	// download with progress
	fmt.Printf("downloading %s", modelName)
	if model.LocalInfo != nil && model.LocalInfo.Size != "" {
		fmt.Printf(" (%s)", model.LocalInfo.Size)
	}
	fmt.Println("...")

	var lastPercent int
	err = whisper.Download(ctx, modelName, func(downloaded, total int64) {
		if total > 0 {
			percent := int(downloaded * 100 / total)
			if percent >= lastPercent+10 {
				fmt.Printf("%d%% ", percent)
				lastPercent = percent
			}
		}
	})
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	path := whisper.GetModelPath(modelName)
	fmt.Printf("\ndownload complete: %s\n", path)
	return nil
}

func modelRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <model-name>",
		Short: "Remove a downloaded local model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelRemove(args[0])
		},
	}
}

func runModelRemove(modelName string) error {
	// find the model across all providers
	model, _, err := provider.FindModelByID(modelName)
	if err != nil {
		return fmt.Errorf("unknown model: %s", modelName)
	}

	// check if it's a cloud model (nothing to remove)
	if !model.NeedsDownload() {
		fmt.Printf("model '%s' is a cloud model, nothing to remove\n", modelName)
		return nil
	}

	// check if installed
	if !whisper.IsInstalled(modelName) {
		return fmt.Errorf("model '%s' is not installed", modelName)
	}

	// remove the model
	if err := whisper.Remove(modelName); err != nil {
		return fmt.Errorf("failed to remove model: %w", err)
	}

	fmt.Printf("model '%s' removed successfully\n", modelName)
	return nil
}
