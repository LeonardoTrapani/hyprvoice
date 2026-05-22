package injection

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type DotoolBackend struct {
	delayMs int64
	holdMs  int64
}

func NewDotoolBackend(delayMs int64, holdMs int64) Backend {
	return &DotoolBackend{delayMs: delayMs, holdMs: holdMs}
}

func (c *DotoolBackend) Name() string {
	return "dotool"
}

func (c *DotoolBackend) Available() error {
	_, err := c.command()
	return err
}

func (c *DotoolBackend) Inject(ctx context.Context, text string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	command, err := c.command()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, command)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	input, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("%s failed: %w", command, err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s failed: %w", command, err)
	}

	if _, err := fmt.Fprint(input, c.commandStream(text)); err != nil {
		input.Close()
		return fmt.Errorf("writing to %s stdin failed: %w", command, err)
	}

	input.Close()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s failed: %w\n%s", command, err, stderr.String())
	}
	if command == "dotool" && stderr.Len() > 0 {
		return fmt.Errorf("%s completed with warnings:\n%s", command, stderr.String())
	}

	return nil
}

func (c *DotoolBackend) command() (string, error) {
	var daemonErr error
	if _, err := exec.LookPath("dotoolc"); err == nil {
		daemonErr = c.dotooldActive()
		if daemonErr == nil {
			return "dotoolc", nil
		}
	}

	if _, err := exec.LookPath("dotool"); err == nil {
		return "dotool", nil
	}

	if daemonErr != nil {
		return "", fmt.Errorf("dotoold unavailable (%v) and dotool not found (install dotool)", daemonErr)
	}
	return "", fmt.Errorf("dotool not found (install dotool)")
}

func (c *DotoolBackend) dotooldActive() error {
	pipe := os.Getenv("DOTOOL_PIPE")
	if pipe == "" {
		pipe = "/tmp/dotool-pipe"
	}

	info, err := os.Lstat(pipe)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("dotoold pipe not found at %s", pipe)
		}
		return fmt.Errorf("dotoold pipe stat failed at %s: %w", pipe, err)
	}
	if err := validateDotoolPipe(pipe, info); err != nil {
		return err
	}

	f, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		if errors.Is(err, syscall.ENXIO) {
			return fmt.Errorf("dotoold not reading pipe at %s", pipe)
		}
		return fmt.Errorf("dotoold pipe open failed at %s: %w", pipe, err)
	}
	if info, err := f.Stat(); err != nil {
		f.Close()
		return fmt.Errorf("dotoold pipe stat failed after open at %s: %w", pipe, err)
	} else if err := validateDotoolPipe(pipe, info); err != nil {
		f.Close()
		return err
	}
	f.Close()

	return nil
}

func validateDotoolPipe(pipe string, info os.FileInfo) error {
	if info.Mode()&os.ModeNamedPipe == 0 {
		return fmt.Errorf("dotoold pipe path is not a FIFO: %s", pipe)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("dotoold pipe stat type unsupported at %s", pipe)
	}
	if int(stat.Uid) != os.Getuid() {
		return fmt.Errorf("dotoold pipe owner mismatch at %s: uid %d", pipe, stat.Uid)
	}

	perm := info.Mode().Perm()
	if perm&0200 == 0 {
		return fmt.Errorf("dotoold pipe is not writable by owner at %s: mode %04o", pipe, perm)
	}
	if perm&0017 != 0 {
		return fmt.Errorf("dotoold pipe has unsafe permissions at %s: mode %04o", pipe, perm)
	}

	return nil
}

func (c *DotoolBackend) commandStream(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	var stream strings.Builder
	fmt.Fprintf(&stream, "typedelay %v\ntypehold %v\n", c.delayMs, c.holdMs)

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			fmt.Fprintf(&stream, "type %s\n", line)
		}
		if i < len(lines)-1 {
			fmt.Fprint(&stream, "key enter\n")
		}
	}

	return stream.String()
}
