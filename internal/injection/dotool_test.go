package injection

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestDotoolMultilineHandling(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		delay    int64
		hold     int64
		wantCmds string
	}{
		{
			name:     "single line",
			text:     "hello",
			delay:    1,
			hold:     2,
			wantCmds: "typedelay 1\ntypehold 2\ntype hello\n",
		},
		{
			name:     "multiline preserves newline with enter",
			text:     "hello\nworld",
			delay:    1,
			hold:     2,
			wantCmds: "typedelay 1\ntypehold 2\ntype hello\nkey enter\ntype world\n",
		},
		{
			name:     "three lines",
			text:     "a\nb\nc",
			delay:    5,
			hold:     10,
			wantCmds: "typedelay 5\ntypehold 10\ntype a\nkey enter\ntype b\nkey enter\ntype c\n",
		},
		{
			name:     "blank line",
			text:     "a\n\nb",
			delay:    1,
			hold:     2,
			wantCmds: "typedelay 1\ntypehold 2\ntype a\nkey enter\nkey enter\ntype b\n",
		},
		{
			name:     "trailing newline",
			text:     "hello\n",
			delay:    1,
			hold:     2,
			wantCmds: "typedelay 1\ntypehold 2\ntype hello\nkey enter\n",
		},
		{
			name:     "crlf newline",
			text:     "hello\r\nworld",
			delay:    1,
			hold:     2,
			wantCmds: "typedelay 1\ntypehold 2\ntype hello\nkey enter\ntype world\n",
		},
		{
			name:     "newline does not inject dotool command",
			text:     "hello\nkey super",
			delay:    1,
			hold:     2,
			wantCmds: "typedelay 1\ntypehold 2\ntype hello\nkey enter\ntype key super\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewDotoolBackend(tt.delay, tt.hold)
			got := captureDotoolStdin(t, backend, tt.text)
			if got != tt.wantCmds {
				t.Errorf("command stream mismatch\nwant:\n%s\ngot:\n%s", tt.wantCmds, got)
			}
		})
	}
}

func TestDotoolConfigurableDelays(t *testing.T) {
	tests := []struct {
		delay int64
		hold  int64
	}{
		{1, 2},
		{5, 10},
		{0, 0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("delay=%d_hold=%d", tt.delay, tt.hold), func(t *testing.T) {
			backend := NewDotoolBackend(tt.delay, tt.hold)
			db := backend.(*DotoolBackend)

			if db.delayMs != tt.delay {
				t.Errorf("delayMs mismatch: want %d, got %d", tt.delay, db.delayMs)
			}
			if db.holdMs != tt.hold {
				t.Errorf("holdMs mismatch: want %d, got %d", tt.hold, db.holdMs)
			}
		})
	}
}

func TestDotoolName(t *testing.T) {
	backend := NewDotoolBackend(1, 2)
	if name := backend.Name(); name != "dotool" {
		t.Errorf("Name() = %q, want %q", name, "dotool")
	}
}

func TestDotoolAvailableCheck(t *testing.T) {
	backend := NewDotoolBackend(1, 2)

	err := backend.Available()
	if err == nil {
		t.Skip("dotool is installed, skipping availability check")
	}
	if !strings.Contains(err.Error(), "dotool") {
		t.Errorf("Expected error mentioning dotool, got: %v", err)
	}
}

func TestDotoolInjectWithFakeCommand(t *testing.T) {
	backend := NewDotoolBackend(1, 2)
	got := captureDotoolStdin(t, backend, "hello\nworld")

	want := "typedelay 1\ntypehold 2\ntype hello\nkey enter\ntype world\n"
	if got != want {
		t.Errorf("captured stdin mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestDotoolUsesClientWhenDaemonPipeIsActive(t *testing.T) {
	tmpDir := t.TempDir()
	stdinFile := filepath.Join(tmpDir, "stdin.txt")
	pipe := filepath.Join(tmpDir, "dotool-pipe")
	if err := syscall.Mkfifo(pipe, 0600); err != nil {
		t.Fatalf("failed to create fifo: %v", err)
	}

	reader, err := os.OpenFile(pipe, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		t.Fatalf("failed to hold fifo reader open: %v", err)
	}
	defer reader.Close()

	writeScript(t, filepath.Join(tmpDir, "dotoolc"), fmt.Sprintf(`#!/bin/sh
/bin/cat > "%s"
exit 0
`, stdinFile))
	writeScript(t, filepath.Join(tmpDir, "dotool"), `#!/bin/sh
echo wrong-backend >&2
exit 42
`)

	t.Setenv("PATH", tmpDir)
	t.Setenv("DOTOOL_PIPE", pipe)

	backend := NewDotoolBackend(1, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := backend.Inject(ctx, "daemon path", 5*time.Second); err != nil {
		t.Fatalf("Inject failed: %v", err)
	}

	gotBytes, err := os.ReadFile(stdinFile)
	if err != nil {
		t.Fatalf("failed to read captured stdin: %v", err)
	}
	want := "typedelay 1\ntypehold 2\ntype daemon path\n"
	if got := string(gotBytes); got != want {
		t.Errorf("captured stdin mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestDotoolFallsBackToDirectCommandWithoutDaemon(t *testing.T) {
	tmpDir := t.TempDir()
	stdinFile := filepath.Join(tmpDir, "stdin.txt")

	writeScript(t, filepath.Join(tmpDir, "dotoolc"), `#!/bin/sh
echo dotoolc should not run >&2
exit 42
`)
	writeScript(t, filepath.Join(tmpDir, "dotool"), fmt.Sprintf(`#!/bin/sh
/bin/cat > "%s"
exit 0
`, stdinFile))

	t.Setenv("PATH", tmpDir)
	t.Setenv("DOTOOL_PIPE", filepath.Join(tmpDir, "missing-pipe"))

	backend := NewDotoolBackend(1, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := backend.Inject(ctx, "direct path", 5*time.Second); err != nil {
		t.Fatalf("Inject failed: %v", err)
	}

	gotBytes, err := os.ReadFile(stdinFile)
	if err != nil {
		t.Fatalf("failed to read captured stdin: %v", err)
	}
	want := "typedelay 1\ntypehold 2\ntype direct path\n"
	if got := string(gotBytes); got != want {
		t.Errorf("captured stdin mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestDotoolDirectCommandWarningsFailForFallback(t *testing.T) {
	tmpDir := t.TempDir()
	stdinFile := filepath.Join(tmpDir, "stdin.txt")

	writeScript(t, filepath.Join(tmpDir, "dotool"), fmt.Sprintf(`#!/bin/sh
/bin/cat > "%s"
echo "impossible character for layout: U+1F41F" >&2
exit 0
`, stdinFile))

	t.Setenv("PATH", tmpDir)
	t.Setenv("DOTOOL_PIPE", filepath.Join(tmpDir, "missing-pipe"))

	backend := NewDotoolBackend(1, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := backend.Inject(ctx, "bad char", 5*time.Second)
	if err == nil {
		t.Fatal("Inject() nil error, want warning error")
	}
	if !strings.Contains(err.Error(), "completed with warnings") {
		t.Fatalf("Inject() error = %v, want warning error", err)
	}
}

func TestDotooldActiveDetection(t *testing.T) {
	backend := NewDotoolBackend(1, 2).(*DotoolBackend)
	tmpDir := t.TempDir()

	t.Run("missing pipe", func(t *testing.T) {
		t.Setenv("DOTOOL_PIPE", filepath.Join(tmpDir, "missing"))
		if err := backend.dotooldActive(); err == nil {
			t.Fatal("dotooldActive() nil error, want missing pipe error")
		}
	})

	t.Run("fifo without reader", func(t *testing.T) {
		pipe := filepath.Join(t.TempDir(), "fifo")
		if err := syscall.Mkfifo(pipe, 0600); err != nil {
			t.Fatalf("failed to create fifo: %v", err)
		}
		t.Setenv("DOTOOL_PIPE", pipe)
		if err := backend.dotooldActive(); err == nil {
			t.Fatal("dotooldActive() nil error, want no reader error")
		}
	})

	t.Run("fifo with unsafe permissions", func(t *testing.T) {
		pipe := filepath.Join(t.TempDir(), "fifo")
		if err := syscall.Mkfifo(pipe, 0600); err != nil {
			t.Fatalf("failed to create fifo: %v", err)
		}
		if err := os.Chmod(pipe, 0666); err != nil {
			t.Fatalf("failed to chmod fifo: %v", err)
		}
		t.Setenv("DOTOOL_PIPE", pipe)
		err := backend.dotooldActive()
		if err == nil {
			t.Fatal("dotooldActive() nil error, want unsafe permissions error")
		}
		if !strings.Contains(err.Error(), "unsafe permissions") {
			t.Fatalf("dotooldActive() error = %v, want unsafe permissions error", err)
		}
	})

	t.Run("fifo with reader", func(t *testing.T) {
		pipe := filepath.Join(t.TempDir(), "fifo")
		if err := syscall.Mkfifo(pipe, 0600); err != nil {
			t.Fatalf("failed to create fifo: %v", err)
		}
		reader, err := os.OpenFile(pipe, os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			t.Fatalf("failed to hold fifo reader open: %v", err)
		}
		defer reader.Close()

		t.Setenv("DOTOOL_PIPE", pipe)
		if err := backend.dotooldActive(); err != nil {
			t.Fatalf("dotooldActive() error = %v, want nil", err)
		}
	})
}

func captureDotoolStdin(t *testing.T, backend Backend, text string) string {
	t.Helper()

	tmpDir := t.TempDir()
	stdinFile := filepath.Join(tmpDir, "stdin.txt")
	fakeDotoolc := filepath.Join(tmpDir, "dotoolc")
	fakeDotool := filepath.Join(tmpDir, "dotool")

	fakeScript := fmt.Sprintf(`#!/bin/sh
/bin/cat > "%s"
exit 0
`, stdinFile)
	writeScript(t, fakeDotoolc, fakeScript)
	writeScript(t, fakeDotool, fakeScript)

	t.Setenv("PATH", tmpDir)
	t.Setenv("DOTOOL_PIPE", filepath.Join(tmpDir, "missing-pipe"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := backend.Inject(ctx, text, 5*time.Second); err != nil {
		t.Fatalf("Inject failed: %v", err)
	}

	got, err := os.ReadFile(stdinFile)
	if err != nil {
		t.Fatalf("failed to read captured stdin: %v", err)
	}

	return string(got)
}

func writeScript(t *testing.T, path, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create %s: %v", path, err)
	}
}
