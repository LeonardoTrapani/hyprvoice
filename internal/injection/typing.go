package injection

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func typeText(ctx context.Context, text string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := checkWtypeAvailable(); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "wtype", text)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wtype failed: %w", err)
	}

	return nil
}

func checkWtypeAvailable() error {
	if _, err := exec.LookPath("wtype"); err != nil {
		return fmt.Errorf("wtype not found: %w (install wtype package)", err)
	}

	// Check for Wayland environment
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("WAYLAND_DISPLAY not set - wtype requires Wayland session")
	}

	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		return fmt.Errorf("XDG_RUNTIME_DIR not set - wtype requires proper session environment")
	}

	return nil
}
