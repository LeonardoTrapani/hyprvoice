package notify

import (
	"testing"
)

func testMessages() map[MessageType]Message {
	return map[MessageType]Message{
		MsgRecordingStarted:   {Title: "Hyprvoice", Body: "Recording Started", IsError: false},
		MsgTranscribing:       {Title: "Hyprvoice", Body: "Transcribing", IsError: false},
		MsgConfigReloaded:     {Title: "Hyprvoice", Body: "Config Reloaded", IsError: false},
		MsgOperationCancelled: {Title: "Hyprvoice", Body: "Operation Cancelled", IsError: false},
		MsgRecordingAborted:   {Title: "", Body: "Recording Aborted", IsError: true},
		MsgInjectionAborted:   {Title: "", Body: "Injection Aborted", IsError: true},
	}
}

func TestDesktop_Send(t *testing.T) {
	desktop := NewDesktop(testMessages())

	// Test Send for different message types (won't actually send, just verify no panic)
	desktop.Send(MsgRecordingStarted)
	desktop.Send(MsgTranscribing)
	desktop.Send(MsgRecordingAborted) // error type
}

func TestDesktop_Error(t *testing.T) {
	desktop := NewDesktop(testMessages())
	desktop.Error("Test Error Message")
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
		name       string
		notifType  string
		expectType string
	}{
		{"desktop", "desktop", "*notify.Desktop"},
		{"log", "log", "*notify.Log"},
		{"none", "none", "*notify.Nop"},
		{"unknown", "unknown", "*notify.Nop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := NewNotifier(tt.notifType, msgs)

			// Test the notifier works
			notifier.Send(MsgRecordingStarted)
			notifier.Error("Error")
		})
	}
}

func TestNotifierInterface(t *testing.T) {
	msgs := testMessages()

	// Test that all notifiers implement the Notifier interface
	var notifier Notifier

	notifier = NewDesktop(msgs)
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
	// Verify MessageDefs contains expected entries
	if len(MessageDefs) != 6 {
		t.Errorf("Expected 6 MessageDefs, got %d", len(MessageDefs))
	}

	// Verify each has required fields
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
	msgs := testMessages()
	desktop := NewDesktop(msgs)

	// Should not panic with unknown message type
	desktop.Send(MessageType(999))
}
