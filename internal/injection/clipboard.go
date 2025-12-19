package injection

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type clipboardBackend struct{}

func NewClipboardBackend() Backend {
	return &clipboardBackend{}
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

	return nil
}
