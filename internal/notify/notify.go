package notify

import (
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

const (
	appName        = "Hyprvoice"
	appIcon        = "audio-input-microphone"
	errorIcon      = "dialog-error"
	expireProgress = 15 * 60 * 1000 // long safety net; Plasma treats 0 as default
	expireDone     = 2500
	expireError    = 5000
	expireDismiss  = 1
)

type Notifier interface {
	Send(mt MessageType)
	Error(msg string) // for dynamic errors (e.g., pipeline errors)
	BeginSession()
	SetMessages(messages map[MessageType]Message)
}

// NewNotifier creates a notifier based on type with resolved messages
func NewNotifier(notifType string, messages map[MessageType]Message) Notifier {
	switch notifType {
	case "desktop":
		return NewDesktop(messages)
	case "log":
		return NewLog(messages)
	default:
		return &Nop{}
	}
}

type notifyRunner func(args []string) (stdout string, err error)

type Desktop struct {
	messages      map[MessageType]Message
	mu            sync.Mutex
	lastID        uint32
	run           notifyRunner
	skipTransient bool
	skipPrintID   bool
}

func NewDesktop(messages map[MessageType]Message) *Desktop {
	return &Desktop{messages: messages, run: runNotifySend}
}

func runNotifySend(args []string) (string, error) {
	cmd := exec.Command("notify-send", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func isProgress(mt MessageType) bool {
	switch mt {
	case MsgRecordingStarted, MsgTranscribing, MsgLLMProcessing:
		return true
	default:
		return false
	}
}

func expireFor(mt MessageType) int {
	if isProgress(mt) {
		return expireProgress
	}
	return expireDone
}

func buildNotifyArgs(title, body, icon, urgency string, expireMS int, replaceID uint32, transient, printID bool) []string {
	args := []string{
		"-a", appName,
		"-i", icon,
		"-u", urgency,
		"-t", strconv.Itoa(expireMS),
	}
	if transient {
		args = append(args, "-e")
	}
	if printID {
		args = append(args, "-p")
	}
	if replaceID != 0 {
		args = append(args, "-r", strconv.FormatUint(uint64(replaceID), 10))
	}
	return append(args, title, body)
}

func normalizeSummary(title, body string) (string, string) {
	if title == "" {
		return appName, body
	}
	return title, body
}

func (d *Desktop) SetMessages(messages map[MessageType]Message) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = messages
}

func (d *Desktop) BeginSession() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dismissLocked()
}

func (d *Desktop) Send(mt MessageType) {
	msg, ok := d.messages[mt]
	if !ok {
		return
	}
	if msg.IsError {
		d.Error(msg.Body)
		return
	}
	d.notify(msg.Title, msg.Body, appIcon, "normal", expireFor(mt), isProgress(mt))
}

func (d *Desktop) Error(msg string) {
	d.notify("Hyprvoice Error", msg, errorIcon, "critical", expireError, false)
}

func (d *Desktop) notify(title, body, icon, urgency string, expireMS int, replace bool) {
	title, body = normalizeSummary(title, body)
	if title == "" {
		title = appName
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	replaceID := uint32(0)
	if replace {
		replaceID = d.lastID
	} else {
		d.dismissLocked()
	}

	args := buildNotifyArgs(title, body, icon, urgency, expireMS, replaceID, !d.skipTransient, !d.skipPrintID)
	out, err := d.exec(args)
	if err != nil {
		log.Printf("Failed to send notification: %v", err)
		d.lastID = 0
		return
	}
	if !replace {
		d.lastID = 0
		return
	}
	if id, parseErr := strconv.ParseUint(out, 10, 32); parseErr == nil {
		d.lastID = uint32(id)
		return
	}
	d.lastID = 0
}

func (d *Desktop) dismissLocked() {
	if d.lastID == 0 {
		return
	}
	args := buildNotifyArgs(appName, "", appIcon, "low", expireDismiss, d.lastID, !d.skipTransient, false)
	_, _ = d.exec(args)
	d.lastID = 0
}

func (d *Desktop) exec(args []string) (string, error) {
	run := d.run
	if run == nil {
		run = runNotifySend
	}

	out, err := run(args)
	if err == nil {
		return out, nil
	}

	if containsFlag(args, "-r") {
		args = stripPair(args, "-r")
		d.lastID = 0
		out, err = run(args)
		if err == nil {
			return out, nil
		}
	}
	if containsFlag(args, "-e") {
		d.skipTransient = true
		args = stripFlag(args, "-e")
		out, err = run(args)
		if err == nil {
			return out, nil
		}
	}
	if containsFlag(args, "-p") {
		d.skipPrintID = true
		args = stripFlag(args, "-p")
		out, err = run(args)
		if err == nil {
			return out, nil
		}
	}
	return "", err
}

func containsFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func stripFlag(args []string, flag string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if arg != flag {
			out = append(out, arg)
		}
	}
	return out
}

func stripPair(args []string, flag string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == flag {
			if i+1 < len(args) {
				i++
			}
			continue
		}
		out = append(out, args[i])
	}
	return out
}

type Log struct {
	mu       sync.Mutex
	messages map[MessageType]Message
}

func NewLog(messages map[MessageType]Message) *Log {
	return &Log{messages: messages}
}

func (l *Log) SetMessages(messages map[MessageType]Message) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = messages
}

func (l *Log) BeginSession() {}

func (l *Log) Send(mt MessageType) {
	l.mu.Lock()
	msg, ok := l.messages[mt]
	l.mu.Unlock()
	if !ok {
		return
	}
	if msg.IsError {
		l.Error(msg.Body)
		return
	}
	log.Printf("%s: %s", msg.Title, msg.Body)
}

func (l *Log) Error(msg string) {
	log.Printf("Hyprvoice Error: %s", msg)
}

type Nop struct{}

func (Nop) Send(mt MessageType)                          {}
func (Nop) Error(msg string)                             {}
func (Nop) BeginSession()                                {}
func (Nop) SetMessages(messages map[MessageType]Message) {}
