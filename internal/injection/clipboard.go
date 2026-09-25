package injection

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// pasteSettleDelay gives wl-copy time to register the selection before the
// paste is requested. Without it a fast target can ask for the clipboard
// before there is anything to hand it.
const pasteSettleDelay = 120 * time.Millisecond

type clipboardBackend struct {
	paste bool
}

func NewClipboardBackend(paste bool) Backend {
	return &clipboardBackend{paste: paste}
}

func (c *clipboardBackend) Name() string {
	return "clipboard"
}

func (c *clipboardBackend) Available() error {
	if _, err := exec.LookPath("wl-copy"); err != nil {
		return fmt.Errorf("wl-copy not found: %w (install wl-clipboard)", err)
	}

	if os.Getenv("WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("WAYLAND_DISPLAY not set - clipboard operations require Wayland session")
	}

	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		return fmt.Errorf("XDG_RUNTIME_DIR not set - clipboard operations require proper session environment")
	}

	return nil
}

func (c *clipboardBackend) Inject(ctx context.Context, text string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := c.Available(); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "wl-copy")
	cmd.Stdin = strings.NewReader(text)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wl-copy failed: %w", err)
	}

	if !c.paste {
		return nil
	}
	return c.sendPaste(ctx)
}

// sendPaste presses Shift+Insert once the text is on the clipboard.
//
// Shift+Insert rather than Ctrl+V because terminals bind Ctrl+V to something
// else, while Shift+Insert pastes in terminals and GUI apps alike.
//
// This is one keystroke against a target that either accepts a paste or does
// nothing, which is why it succeeds where synthesised typing does not: keys
// that miss a text field become application shortcuts, a paste that misses one
// is simply ignored.
func (c *clipboardBackend) sendPaste(ctx context.Context) error {
	if _, err := exec.LookPath("wtype"); err != nil {
		return fmt.Errorf("clipboard_paste needs wtype: %w", err)
	}

	// The clipboard offer must be live before the paste is requested.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(pasteSettleDelay):
	}

	cmd := exec.CommandContext(ctx, "wtype", "-M", "shift", "-k", "Insert", "-m", "shift")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("paste keystroke failed: %w", err)
	}

	return nil
}
