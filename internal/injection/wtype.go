package injection

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type wtypeBackend struct {
	startDelay time.Duration
	keyDelay   time.Duration
}

func NewWtypeBackend(startDelay, keyDelay time.Duration) Backend {
	return &wtypeBackend{startDelay: startDelay, keyDelay: keyDelay}
}

func (w *wtypeBackend) Name() string {
	return "wtype"
}

func (w *wtypeBackend) Available() error {
	if _, err := exec.LookPath("wtype"); err != nil {
		return fmt.Errorf("wtype not found: %w (install wtype package)", err)
	}

	if os.Getenv("WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("WAYLAND_DISPLAY not set - wtype requires Wayland session")
	}

	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		return fmt.Errorf("XDG_RUNTIME_DIR not set - wtype requires proper session environment")
	}

	return nil
}

func (w *wtypeBackend) Inject(ctx context.Context, text string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := w.Available(); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "wtype", w.args(text)...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wtype failed: %w", err)
	}

	return nil
}

// args builds the wtype invocation, applying the configured delays.
//
// -s pauses before the following options are interpreted, which is what gives
// a Chromium or Electron target time to adopt the keymap wtype just uploaded;
// -d spaces out the individual keystrokes.
func (w *wtypeBackend) args(text string) []string {
	var args []string

	if ms := int(w.startDelay.Milliseconds()); ms > 0 {
		args = append(args, "-s", strconv.Itoa(ms))
	}
	if ms := int(w.keyDelay.Milliseconds()); ms > 0 {
		args = append(args, "-d", strconv.Itoa(ms))
	}

	return append(args, "--", text)
}
