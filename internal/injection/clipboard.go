package injection

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func getClipboard(ctx context.Context, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wl-paste", "--no-newline")
	output, err := cmd.Output()
	if err != nil {
		return "", nil
	}

	return string(output), nil
}

func setClipboard(ctx context.Context, text string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wl-copy")
	cmd.Stdin = strings.NewReader(text)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wl-copy failed: %w", err)
	}

	return nil
}

func checkClipboardAvailable() error {
	if _, err := exec.LookPath("wl-copy"); err != nil {
		return fmt.Errorf("wl-copy not found: %w (install wl-clipboard)", err)
	}

	if _, err := exec.LookPath("wl-paste"); err != nil {
		return fmt.Errorf("wl-paste not found: %w (install wl-clipboard)", err)
	}

	// Check for Wayland environment
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("WAYLAND_DISPLAY not set - clipboard operations require Wayland session")
	}

	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		return fmt.Errorf("XDG_RUNTIME_DIR not set - clipboard operations require proper session environment")
	}

	return nil
}
