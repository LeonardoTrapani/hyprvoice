package notify

import (
	"log"
	"os/exec"
)

type Notifier interface {
	Error(msg string)
	Notify(title, message string)
}

type Desktop struct{}

func (d Desktop) RecordingStarted() {
	d.Notify("Hyprvoice", "Recording Started")
}

func (d Desktop) Transcribing() {
	d.Notify("Hyprvoice", "Transcribing...")
}

func (Desktop) Error(msg string) {
	cmd := exec.Command("notify-send", "-a", "Hyprvoice", "-u", "critical", "Hyprvoice Error", msg)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send error notification: %v", err)
	}
}

func (Desktop) Notify(title, message string) {
	cmd := exec.Command("notify-send", "-a", "Hyprvoice", title, message)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

type Log struct{}

func (l Log) Error(msg string) {
	l.Notify("Hyprvoice Error", msg)
}

func (Log) Notify(title, message string) {
	log.Printf("%s: %s", title, message)
}

type Nop struct{}

func (Nop) Error(msg string)             {}
func (Nop) Notify(title, message string) {}
