package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/history"
	"github.com/spf13/cobra"
)

// openHistory resolves the store from config, falling back to defaults when
// there is no usable config yet -- reading the archive should not require one.
func openHistory() (*history.Store, error) {
	path, maxEntries := "", 0

	if conf, err := config.Load(); err == nil && conf != nil {
		path, maxEntries = conf.History.Path, conf.History.MaxEntries
	}

	if path == "" {
		var err error
		if path, err = history.DefaultPath(); err != nil {
			return nil, fmt.Errorf("resolve history path: %w", err)
		}
	}
	return history.New(path, maxEntries), nil
}

func historyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Browse past transcriptions",
		Long: "Browse past transcriptions.\n\n" +
			"Dictated text is typed into whichever window had focus and is gone if\n" +
			"that window did not want it. hyprvoice keeps a rolling archive so the\n" +
			"transcription itself survives a mis-aimed or failed injection.",

		// Loading the config logs as a side effect. These commands are meant
		// to be piped (`hyprvoice history last | wl-copy`), so keep the
		// daemon's chatter out of them.
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.SetOutput(io.Discard)
		},
	}

	cmd.AddCommand(historyListCmd(), historyLastCmd(), historyGetCmd(), historyClearCmd())
	return cmd
}

func historyListCmd() *cobra.Command {
	var limit int
	var format string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent transcriptions, newest first",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openHistory()
			if err != nil {
				return err
			}

			entries, err := store.List(limit)
			if err != nil {
				return fmt.Errorf("read history: %w", err)
			}

			if format == "json" {
				// Always an array, never null, so consumers can iterate
				// without a nil check.
				if entries == nil {
					entries = []history.Entry{}
				}
				return json.NewEncoder(cmd.OutOrStdout()).Encode(entries)
			}

			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No transcriptions recorded yet.")
				return nil
			}

			for _, e := range entries {
				fmt.Fprintf(cmd.OutOrStdout(), "%s  %s  %s\n",
					e.At.Local().Format("2006-01-02 15:04"), statusMark(e), oneLine(e.Text))
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "how many to show; 0 for all")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}

func historyLastCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "last",
		Short: "Print the most recent transcription",
		Long: "Print the most recent transcription.\n\n" +
			"Prints the text alone, so it can be piped straight back:\n" +
			"  hyprvoice history last | wl-copy",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openHistory()
			if err != nil {
				return err
			}

			entry, ok, err := store.Last()
			if err != nil {
				return fmt.Errorf("read history: %w", err)
			}
			if !ok {
				return fmt.Errorf("no transcriptions recorded yet")
			}

			if format == "json" {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(entry)
			}
			fmt.Fprintln(cmd.OutOrStdout(), entry.Text)
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}

func historyGetCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Print one transcription by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openHistory()
			if err != nil {
				return err
			}

			entry, ok, err := store.Get(args[0])
			if err != nil {
				return fmt.Errorf("read history: %w", err)
			}
			if !ok {
				return fmt.Errorf("no transcription with id %q", args[0])
			}

			if format == "json" {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(entry)
			}
			fmt.Fprintln(cmd.OutOrStdout(), entry.Text)
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}

func historyClearCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Delete every recorded transcription",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("this deletes every recorded transcription; pass --yes to confirm")
			}

			store, err := openHistory()
			if err != nil {
				return err
			}
			if err := store.Clear(); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "History cleared.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return cmd
}

// statusMark flags the entries worth noticing: text that never reached a
// window, and text the LLM pass rewrote.
func statusMark(e history.Entry) string {
	switch {
	case !e.Injected:
		return "!"
	case e.Raw != "":
		return "~"
	default:
		return " "
	}
}

// oneLine renders a transcription as a single bounded line for the list view.
func oneLine(text string) string {
	const width = 100

	flat := strings.Join(strings.Fields(text), " ")
	if len(flat) <= width {
		return flat
	}

	// Trim on a rune boundary so a multi-byte character is not cut in half.
	runes := []rune(flat)
	if len(runes) <= width {
		return flat
	}
	return string(runes[:width-1]) + "…"
}
