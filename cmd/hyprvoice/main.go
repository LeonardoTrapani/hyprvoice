package main

import (
	"fmt"

	"github.com/leonardotrapani/hyprvoice/internal/bus"
	"github.com/leonardotrapani/hyprvoice/internal/hotkeydaemon"
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
	)
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the hotkey daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return (hotkeydaemon.New(nil)).Run()
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
				return fmt.Errorf("failed to toggle: %w", err)
			}
			fmt.Print(resp) // print STATUS line
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
			fmt.Print(resp) // print STATUS line
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
			fmt.Print(resp) // print STATUS proto= line
			return nil
		},
	}
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the hotkey daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := bus.SendCommand('q')
			if err != nil {
				return fmt.Errorf("failed to stop daemon: %w", err)
			}
			fmt.Print(resp) // print OK quitting
			return nil
		},
	}
}
