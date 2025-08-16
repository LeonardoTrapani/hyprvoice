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
	Notify(title, message string)
}

type Desktop struct{}

func (d Desktop) RecordingStarted() {
	d.Notify("Hyprvoice", "Recording Started")
}

func (d Desktop) RecordingEnded() {
	d.Notify("Hyprvoice", "Recording Ended")
}

func (d Desktop) Transcribing() {
	d.Notify("Hyprvoice", "Transcribing...")
}

func (Desktop) Aborted() {
	cmd := exec.Command("notify-send", "-a", "Hyprvoice", "-u", "critical", "Hyprvoice: Aborted")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send abort notification: %v", err)
	}
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

func (l Log) RecordingStarted() {
	l.Notify("Hyprvoice", "Recording Started")
}

func (l Log) RecordingEnded() {
	l.Notify("Hyprvoice", "Recording Ended")
}

func (l Log) Transcribing() {
	l.Notify("Hyprvoice", "Transcribing...")
}

func (l Log) Aborted() {
	l.Notify("Hyprvoice", "Aborted")
}

func (l Log) Error(msg string) {
	l.Notify("Hyprvoice Error", msg)
}

func (Log) Notify(title, message string) {
	log.Printf("%s: %s", title, message)
}

type Nop struct{}

func (Nop) RecordingStarted()            {}
func (Nop) RecordingEnded()              {}
func (Nop) Aborted()                     {}
func (Nop) Transcribing()                {}
func (Nop) Error(msg string)             {}
func (Nop) Notify(title, message string) {}
