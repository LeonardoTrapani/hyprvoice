package notify

import (
	"fmt"
	"log"
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
	cmd := exec.Command("notify-send", "-a", "Hyprvoice",
		fmt.Sprintf("Hyprvoice: %s Recording", state))
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

func (Desktop) Error(msg string) {
	cmd := exec.Command("notify-send", "-a", "Hyprvoice", "-u", "critical", msg)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send error notification: %v", err)
	}
}

// Nop is a Notifier that does absolutely nothing.
// Useful in unit tests or headless builds.
type Nop struct{}

func (Nop) RecordingChanged(on bool) {}
func (Nop) Error(msg string)         {}
