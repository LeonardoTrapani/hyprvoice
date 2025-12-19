package injection

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
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

	// Check if ydotoold is running by checking socket
	socketPath := y.getSocketPath()
	if socketPath == "" {
		return fmt.Errorf("ydotoold socket not found - ensure ydotoold is running")
	}

	// Try to connect to verify daemon is responsive
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		return fmt.Errorf("ydotoold not responding at %s: %w", socketPath, err)
	}
	conn.Close()

	return nil
}

func (y *ydotoolBackend) getSocketPath() string {
	// Check YDOTOOL_SOCKET env var first
	if sock := os.Getenv("YDOTOOL_SOCKET"); sock != "" {
		if _, err := os.Stat(sock); err == nil {
			return sock
		}
	}

	// Check common locations
	paths := []string{
		"/run/user/" + fmt.Sprint(os.Getuid()) + "/.ydotool_socket",
		"/tmp/.ydotool_socket",
	}

	// Also check XDG_RUNTIME_DIR
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		paths = append([]string{filepath.Join(xdg, ".ydotool_socket")}, paths...)
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
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
