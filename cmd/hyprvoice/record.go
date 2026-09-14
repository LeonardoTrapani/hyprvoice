package main

import (
	"fmt"

	"github.com/leonardotrapani/hyprvoice/internal/bus"
	"github.com/spf13/cobra"
)

func recordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "record",
		Short: "Start and stop recording explicitly",
		Long: "Start and stop recording explicitly.\n\n" +
			"`toggle` flips whatever state the daemon is in, which is right for a\n" +
			"tap-on/tap-off key but wrong for push-to-talk: binding it to both press\n" +
			"and release means a release the compositor never delivers leaves the\n" +
			"daemon recording AND inverts every press that follows.\n\n" +
			"These two are idempotent. Pressing start while already recording does\n" +
			"nothing, and stopping when nothing is recording does nothing, so a lost\n" +
			"edge costs one dictation instead of every one after it.",
	}

	cmd.AddCommand(recordStartCmd(), recordStopCmd())
	return cmd
}

func recordStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Begin recording, or do nothing if already recording",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := bus.SendCommand('r')
			if err != nil {
				return fmt.Errorf("failed to start recording: %w", err)
			}

			fmt.Fprint(cmd.OutOrStdout(), resp)
			return nil
		},
	}
}

func recordStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Finish recording and transcribe, or do nothing if not recording",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := bus.SendCommand('f')
			if err != nil {
				return fmt.Errorf("failed to stop recording: %w", err)
			}

			fmt.Fprint(cmd.OutOrStdout(), resp)
			return nil
		},
	}
}
