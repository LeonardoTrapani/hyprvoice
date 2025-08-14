package notify

import (
	"log"
	"os/exec"
)

type Notifier interface {
	RecordingStarted()
	RecordingEnded()
	Aborted()
	Transcribing()
	Error(msg string)
}

type Desktop struct{}

func (Desktop) RecordingStarted() {
	cmd := exec.Command("notify-send", "-a", "Hyprvoice", "Hyprvoice: Recording Started")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

func (Desktop) RecordingEnded() {
	cmd := exec.Command("notify-send", "-a", "Hyprvoice", "Hyprvoice: Recording Ended")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

func (Desktop) Transcribing() {
	cmd := exec.Command("notify-send", "-a", "Hyprvoice", "Hyprvoice: Transcribing...")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

func (Desktop) Aborted() {
	cmd := exec.Command("notify-send", "-a", "Hyprvoice", "-u", "critical", "Hyprvoice: Aborted")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send abort notification: %v", err)
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

func (Nop) RecordingStarted() {}
func (Nop) RecordingEnded()   {}
func (Nop) Transcribing()     {}
func (Nop) Error(msg string)  {}
