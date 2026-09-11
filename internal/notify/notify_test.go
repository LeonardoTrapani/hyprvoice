package notify

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func testMessages() map[MessageType]Message {
	return map[MessageType]Message{
		MsgRecordingStarted:   {Title: "Hyprvoice", Body: "Recording Started", IsError: false},
		MsgTranscribing:       {Title: "Hyprvoice", Body: "Transcribing", IsError: false},
		MsgLLMProcessing:      {Title: "Hyprvoice", Body: "Processing...", IsError: false},
		MsgInjectionComplete:  {Title: "Hyprvoice", Body: "Ready", IsError: false},
		MsgConfigReloaded:     {Title: "Hyprvoice", Body: "Config Reloaded", IsError: false},
		MsgOperationCancelled: {Title: "Hyprvoice", Body: "Operation Cancelled", IsError: false},
		MsgRecordingAborted:   {Title: "", Body: "Recording Aborted", IsError: true},
		MsgInjectionAborted:   {Title: "", Body: "Injection Aborted", IsError: true},
	}
}

func TestLog_Send(t *testing.T) {
	logNotifier := NewLog(testMessages())

	logNotifier.Send(MsgRecordingStarted)
	logNotifier.Send(MsgRecordingAborted) // error type
}

func TestLog_Error(t *testing.T) {
	logNotifier := NewLog(testMessages())
	logNotifier.Error("Test Error Message")
}

func TestNop_Send(t *testing.T) {
	nop := Nop{}
	nop.Send(MsgRecordingStarted)
	nop.Send(MsgRecordingAborted)
}

func TestNop_Error(t *testing.T) {
	nop := Nop{}
	nop.Error("Test Error Message")
}

func TestNewNotifier(t *testing.T) {
	msgs := testMessages()

	tests := []struct {
		name      string
		notifType string
	}{
		{"desktop", "desktop"},
		{"log", "log"},
		{"none", "none"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := NewNotifier(tt.notifType, msgs)
			if notifier == nil {
				t.Fatal("NewNotifier returned nil")
			}
			notifier.BeginSession()
			notifier.SetMessages(msgs)
		})
	}
}

func TestNotifierInterface(t *testing.T) {
	msgs := testMessages()

	var notifier Notifier

	desktop := NewDesktop(msgs)
	desktop.run = func(args []string) (string, error) { return "1", nil }
	notifier = desktop
	notifier.Send(MsgRecordingStarted)
	notifier.Error("Error")

	notifier = NewLog(msgs)
	notifier.Send(MsgRecordingStarted)
	notifier.Error("Error")

	notifier = &Nop{}
	notifier.Send(MsgRecordingStarted)
	notifier.Error("Error")
}

func TestMessageDefs(t *testing.T) {
	if len(MessageDefs) != 8 {
		t.Errorf("Expected 8 MessageDefs, got %d", len(MessageDefs))
	}

	for _, def := range MessageDefs {
		if def.ConfigKey == "" {
			t.Errorf("MessageDef type %d has empty ConfigKey", def.Type)
		}
		if def.DefaultBody == "" {
			t.Errorf("MessageDef type %d has empty DefaultBody", def.Type)
		}
	}
}

func TestSend_UnknownMessageType(t *testing.T) {
	desktop := NewDesktop(testMessages())
	called := false
	desktop.run = func(args []string) (string, error) {
		called = true
		return "1", nil
	}

	desktop.Send(MessageType(999))
	if called {
		t.Error("unknown message type should not call notify-send")
	}
}

func TestBuildNotifyArgs(t *testing.T) {
	args := buildNotifyArgs("Hyprvoice", "Ready", appIcon, "normal", expireDone, 7, true, true)
	got := strings.Join(args, " ")
	wantParts := []string{
		"-a " + appName,
		"-i " + appIcon,
		"-u normal",
		"-e",
		"-p",
		"-t " + strconv.Itoa(expireDone),
		"-r 7",
		"Hyprvoice Ready",
	}
	for _, part := range wantParts {
		if !strings.Contains(got, part) {
			t.Errorf("buildNotifyArgs() = %q, missing %q", got, part)
		}
	}
}

func TestExpireFor(t *testing.T) {
	if expireFor(MsgRecordingStarted) != expireProgress {
		t.Errorf("recording expire = %d, want %d", expireFor(MsgRecordingStarted), expireProgress)
	}
	if expireFor(MsgTranscribing) != expireProgress {
		t.Errorf("transcribing expire = %d, want %d", expireFor(MsgTranscribing), expireProgress)
	}
	if expireFor(MsgLLMProcessing) != expireProgress {
		t.Errorf("processing expire = %d, want %d", expireFor(MsgLLMProcessing), expireProgress)
	}
	if expireFor(MsgInjectionComplete) != expireDone {
		t.Errorf("ready expire = %d, want %d", expireFor(MsgInjectionComplete), expireDone)
	}
	if expireFor(MsgOperationCancelled) != expireDone {
		t.Errorf("cancelled expire = %d, want %d", expireFor(MsgOperationCancelled), expireDone)
	}
}

func TestNormalizeSummary(t *testing.T) {
	title, body := normalizeSummary("", "🎙️")
	if title != appName || body != "🎙️" {
		t.Errorf("normalizeSummary(empty title) = (%q, %q), want body kept under default title", title, body)
	}

	title, body = normalizeSummary("", "")
	if title != appName || body != "" {
		t.Errorf("normalizeSummary(empty title and body) = (%q, %q)", title, body)
	}

	title, body = normalizeSummary("Hyprvoice", "Ready")
	if title != "Hyprvoice" || body != "Ready" {
		t.Errorf("normalizeSummary() = (%q, %q)", title, body)
	}
}

func TestDesktop_ReplacesProgressAndDismissesReady(t *testing.T) {
	desktop := NewDesktop(testMessages())
	var calls [][]string
	nextID := 10
	desktop.run = func(args []string) (string, error) {
		copied := append([]string(nil), args...)
		calls = append(calls, copied)
		id := nextID
		nextID++
		return strconv.Itoa(id), nil
	}

	desktop.Send(MsgRecordingStarted)
	desktop.Send(MsgTranscribing)
	desktop.Send(MsgInjectionComplete)

	if len(calls) != 4 {
		t.Fatalf("expected 4 notify-send calls (start, transcribe, dismiss, ready), got %d", len(calls))
	}
	if !containsPair(calls[0], "-t", strconv.Itoa(expireProgress)) {
		t.Errorf("recording notification should use a long expire, args=%v", calls[0])
	}
	if !containsPair(calls[0], "-i", appIcon) {
		t.Errorf("recording notification missing icon, args=%v", calls[0])
	}
	if containsFlag(calls[0], "-r") {
		t.Errorf("first notification should not replace, args=%v", calls[0])
	}
	if !containsPair(calls[1], "-r", "10") {
		t.Errorf("transcribing should replace prior id 10, args=%v", calls[1])
	}
	if !containsPair(calls[2], "-r", "11") || !containsPair(calls[2], "-t", strconv.Itoa(expireDismiss)) {
		t.Errorf("ready should dismiss prior in-progress toast, args=%v", calls[2])
	}
	if containsFlag(calls[3], "-r") {
		t.Errorf("ready should be a new toast so Plasma does not keep the old image, args=%v", calls[3])
	}
	if !containsPair(calls[3], "-t", strconv.Itoa(expireDone)) {
		t.Errorf("ready notification should expire, args=%v", calls[3])
	}
	if desktop.lastID != 0 {
		t.Errorf("lastID = %d, want 0 after terminal toast", desktop.lastID)
	}
}

func TestDesktop_ErrorUsesDistinctIcon(t *testing.T) {
	desktop := NewDesktop(testMessages())
	var args []string
	desktop.run = func(got []string) (string, error) {
		args = append([]string(nil), got...)
		return "3", nil
	}

	desktop.Send(MsgRecordingAborted)
	if !containsPair(args, "-i", errorIcon) {
		t.Errorf("error notification missing error icon, args=%v", args)
	}
	if !containsPair(args, "-u", "critical") {
		t.Errorf("error notification should be critical, args=%v", args)
	}
	if !containsPair(args, "-t", strconv.Itoa(expireError)) {
		t.Errorf("error notification should expire, args=%v", args)
	}
}

func TestDesktop_RetriesWithoutStaleReplaceID(t *testing.T) {
	desktop := NewDesktop(testMessages())
	desktop.lastID = 99
	var calls [][]string
	desktop.run = func(args []string) (string, error) {
		calls = append(calls, append([]string(nil), args...))
		if containsFlag(args, "-r") {
			return "", errors.New("stale id")
		}
		return "4", nil
	}

	desktop.Send(MsgTranscribing)
	if len(calls) != 2 {
		t.Fatalf("expected retry after stale replace-id, got %d calls", len(calls))
	}
	if containsFlag(calls[1], "-r") {
		t.Errorf("retry should drop replace-id, args=%v", calls[1])
	}
	if desktop.lastID != 4 {
		t.Errorf("lastID = %d, want 4", desktop.lastID)
	}
}

func TestDesktop_NonNumericPrintIDClearsLastID(t *testing.T) {
	desktop := NewDesktop(testMessages())
	desktop.run = func(args []string) (string, error) {
		return "not-an-id", nil
	}

	desktop.Send(MsgRecordingStarted)
	if desktop.lastID != 0 {
		t.Errorf("lastID = %d, want 0 when print-id is not numeric", desktop.lastID)
	}
}

func TestDesktop_RetriesWithoutUnsupportedTransient(t *testing.T) {
	desktop := NewDesktop(testMessages())
	var calls [][]string
	desktop.run = func(args []string) (string, error) {
		calls = append(calls, append([]string(nil), args...))
		if containsFlag(args, "-e") {
			return "", errors.New("unknown option")
		}
		return "8", nil
	}

	desktop.Send(MsgRecordingStarted)
	if !desktop.skipTransient {
		t.Error("expected skipTransient after -e failure")
	}
	if desktop.lastID != 8 {
		t.Errorf("lastID = %d, want 8", desktop.lastID)
	}

	desktop.Send(MsgTranscribing)
	if containsFlag(calls[len(calls)-1], "-e") {
		t.Errorf("later sends should omit unsupported -e, args=%v", calls[len(calls)-1])
	}
}

func TestDesktop_BeginSessionDismissesActive(t *testing.T) {
	desktop := NewDesktop(testMessages())
	var calls [][]string
	desktop.lastID = 21
	desktop.run = func(args []string) (string, error) {
		calls = append(calls, append([]string(nil), args...))
		return "1", nil
	}

	desktop.BeginSession()
	if desktop.lastID != 0 {
		t.Errorf("lastID = %d, want 0 after BeginSession", desktop.lastID)
	}
	if len(calls) != 1 || !containsPair(calls[0], "-r", "21") {
		t.Errorf("BeginSession should dismiss id 21, calls=%v", calls)
	}
}

func containsPair(args []string, flag, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}
