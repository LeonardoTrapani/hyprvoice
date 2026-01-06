package injection

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

type ydotoolBackend struct{}

func NewYdotoolBackend() Backend {
	return &ydotoolBackend{}
}

func (y *ydotoolBackend) Name() string {
	return "ydotool"
}

func (y *ydotoolBackend) Available() error {
	if _, err := exec.LookPath("ydotool"); err != nil {
		return fmt.Errorf("ydotool not found: %w (install ydotool package)", err)
	}
	return nil
}

func (y *ydotoolBackend) Inject(ctx context.Context, text string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := y.Available(); err != nil {
		return err
	}

	// ydotool type -- "text"
	cmd := exec.CommandContext(ctx, "ydotool", "type", "--", text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ydotool failed: %w", err)
	}

	return nil
}
