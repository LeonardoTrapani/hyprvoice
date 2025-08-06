package notify

import (
	"fmt"
	"os/exec"
)

type Notifier interface {
	RecordingChanged(on bool)
	Error(msg string)
}

type Desktop struct{}

func (Desktop) RecordingChanged(on bool) {
	state := "Stopped"
	if on {
		state = "Started"
	}
	exec.Command("notify-send", "-a", "Hyprvoice",
		fmt.Sprintf("Hyprvoice: %s Recording", state)).Run()
}

func (Desktop) Error(msg string) {
	exec.Command("notify-send", "-a", "Hyprvoice", "-u", "critical", msg).Run()
}

// Nop is a Notifier that does absolutely nothing.
// Useful in unit tests or headless builds.
type Nop struct{}

func (Nop) RecordingChanged(on bool) {}
func (Nop) Error(msg string)         {}
