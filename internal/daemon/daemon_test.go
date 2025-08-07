package daemon

import (
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/bus"
	"github.com/leonardotrapani/hyprvoice/internal/notify"
)

func TestToggle(t *testing.T) {
	// Clean up any existing daemon
	bus.RemovePidFile()

	d := New(notify.Nop{})

	// Start daemon in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run()
	}()

	// Wait for daemon to be ready by trying to connect
	maxAttempts := 50
	for i := range maxAttempts {
		if _, err := bus.SendCommand('s'); err == nil {
			break // daemon is ready
		}
		if i == maxAttempts-1 {
			t.Fatal("daemon failed to start within timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}

	defer func() {
		bus.SendCommand('q')
		// Wait for daemon to exit
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
			t.Error("daemon did not exit within timeout")
		}
	}()

	// Test first toggle
	if out, err := bus.SendCommand('t'); err != nil {
		t.Fatalf("first toggle failed: %v", err)
	} else if out != "STATUS recording=true\n" {
		t.Fatalf("unexpected first toggle response: %s", out)
	}

	if !d.Rec() {
		t.Fatalf("state should be true after first toggle")
	}

	// Test second toggle
	if out, err := bus.SendCommand('t'); err != nil {
		t.Fatalf("second toggle failed: %v", err)
	} else if out != "STATUS recording=false\n" {
		t.Fatalf("unexpected second toggle response: %s", out)
	}

	if d.Rec() {
		t.Fatalf("state should be false after second toggle")
	}
}
