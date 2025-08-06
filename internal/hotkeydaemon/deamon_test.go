package hotkeydaemon

import (
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/bus"
	"github.com/leonardotrapani/hyprvoice/internal/notify"
)

func TestToggle(t *testing.T) {
	d := New(notify.Nop{})
	go d.Run()
	time.Sleep(50 * time.Millisecond) // give listener time to start
	defer bus.SendCommand('q')

	if out, _ := bus.SendCommand('t'); out != "STATUS recording=true\n" {
		t.Fatalf("unexpected: %s", out)
	}
	if !d.Rec() {
		t.Fatalf("state should be true after first toggle")
	}
	if out, _ := bus.SendCommand('t'); out != "STATUS recording=false\n" {
		t.Fatalf("unexpected: %s", out)
	}
	if d.Rec() {
		t.Fatalf("state should be false after second toggle")
	}
}
