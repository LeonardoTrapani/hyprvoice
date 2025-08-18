package injection

import (
	"context"
	"fmt"
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

	return nil
}
