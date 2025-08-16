package notify

import (
	"bytes"
	"log"
	"os"
	"testing"
)

func TestDesktopNotifier(t *testing.T) {
	desktop := Desktop{}

	t.Run("RecordingStarted", func(t *testing.T) {
		// This test will actually try to call notify-send if available
		// We can't easily mock exec.Command, so we just verify it doesn't panic
		desktop.RecordingStarted()
	})

	t.Run("RecordingEnded", func(t *testing.T) {
		desktop.RecordingEnded()
	})

	t.Run("Transcribing", func(t *testing.T) {
		desktop.Transcribing()
	})

	t.Run("Aborted", func(t *testing.T) {
		desktop.Aborted()
	})

	t.Run("Error", func(t *testing.T) {
		desktop.Error("test error message")
	})

	t.Run("Notify", func(t *testing.T) {
		desktop.Notify("Test Title", "Test Message")
	})
}

func TestLogNotifier(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	logNotifier := Log{}

	t.Run("RecordingStarted", func(t *testing.T) {
		buf.Reset()
		logNotifier.RecordingStarted()

		output := buf.String()
		if output == "" {
			t.Error("should log recording started message")
		}
		if !containsSubstring(output, "Hyprvoice") || !containsSubstring(output, "Recording Started") {
			t.Errorf("log output should contain expected message, got: %s", output)
		}
	})

	t.Run("RecordingEnded", func(t *testing.T) {
		buf.Reset()
		logNotifier.RecordingEnded()

		output := buf.String()
		if !containsSubstring(output, "Recording Ended") {
			t.Errorf("log output should contain 'Recording Ended', got: %s", output)
		}
	})

	t.Run("Transcribing", func(t *testing.T) {
		buf.Reset()
		logNotifier.Transcribing()

		output := buf.String()
		if !containsSubstring(output, "Transcribing") {
			t.Errorf("log output should contain 'Transcribing', got: %s", output)
		}
	})

	t.Run("Aborted", func(t *testing.T) {
		buf.Reset()
		logNotifier.Aborted()

		output := buf.String()
		if !containsSubstring(output, "Aborted") {
			t.Errorf("log output should contain 'Aborted', got: %s", output)
		}
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		testMsg := "test error message"
		logNotifier.Error(testMsg)

		output := buf.String()
		if !containsSubstring(output, "Hyprvoice Error") || !containsSubstring(output, testMsg) {
			t.Errorf("log output should contain error message, got: %s", output)
		}
	})

	t.Run("Notify", func(t *testing.T) {
		buf.Reset()
		title := "Test Title"
		message := "Test Message"
		logNotifier.Notify(title, message)

		output := buf.String()
		if !containsSubstring(output, title) || !containsSubstring(output, message) {
			t.Errorf("log output should contain title and message, got: %s", output)
		}
	})
}

func TestNopNotifier(t *testing.T) {
	nop := Nop{}

	// All Nop methods should do nothing and not panic
	t.Run("all methods should not panic", func(t *testing.T) {
		nop.RecordingStarted()
		nop.RecordingEnded()
		nop.Transcribing()
		nop.Aborted()
		nop.Error("test message")
		nop.Notify("title", "message")
	})
}

func TestNotifierInterface(t *testing.T) {
	// Verify all types implement the Notifier interface
	var notifiers []Notifier = []Notifier{
		Desktop{},
		Log{},
		Nop{},
	}

	for i, notifier := range notifiers {
		t.Run("interface compliance", func(t *testing.T) {
			// Test that all interface methods can be called
			notifier.RecordingStarted()
			notifier.RecordingEnded()
			notifier.Transcribing()
			notifier.Aborted()
			notifier.Error("test")
			notifier.Notify("title", "message")
		})

		// Verify the notifier is not nil
		if notifier == nil {
			t.Errorf("notifier %d should not be nil", i)
		}
	}
}

func TestNotifierBehaviorConsistency(t *testing.T) {
	// Test that different notifiers handle the same inputs consistently
	testCases := []struct {
		name    string
		title   string
		message string
	}{
		{"empty strings", "", ""},
		{"normal strings", "Title", "Message"},
		{"unicode strings", "📢 Alert", "🎙️ Recording"},
		{"long strings", "Very Long Title That Might Cause Issues", "Very long message that contains a lot of text and might cause formatting issues or truncation in some notification systems"},
		{"special characters", "Title with \n newlines", "Message with \"quotes\" and 'apostrophes'"},
	}

	notifiers := map[string]Notifier{
		"Desktop": Desktop{},
		"Log":     Log{},
		"Nop":     Nop{},
	}

	for notifierName, notifier := range notifiers {
		for _, tc := range testCases {
			t.Run(notifierName+"_"+tc.name, func(t *testing.T) {
				// These should not panic regardless of input
				notifier.Notify(tc.title, tc.message)
				notifier.Error(tc.message)
			})
		}
	}
}

func TestLogNotifierOutput(t *testing.T) {
	// More detailed testing of log output format
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	logNotifier := Log{}

	t.Run("log format consistency", func(t *testing.T) {
		testCases := []struct {
			method   func()
			expected []string
		}{
			{
				method:   logNotifier.RecordingStarted,
				expected: []string{"Hyprvoice", "Recording Started"},
			},
			{
				method:   logNotifier.RecordingEnded,
				expected: []string{"Hyprvoice", "Recording Ended"},
			},
			{
				method:   logNotifier.Transcribing,
				expected: []string{"Hyprvoice", "Transcribing"},
			},
			{
				method:   logNotifier.Aborted,
				expected: []string{"Hyprvoice", "Aborted"},
			},
		}

		for _, tc := range testCases {
			buf.Reset()
			tc.method()
			output := buf.String()

			for _, expected := range tc.expected {
				if !containsSubstring(output, expected) {
					t.Errorf("log output should contain %q, got: %s", expected, output)
				}
			}
		}
	})

	t.Run("error method with different messages", func(t *testing.T) {
		errorMessages := []string{
			"simple error",
			"error with numbers 123",
			"error with symbols !@#$%",
			"",
		}

		for _, msg := range errorMessages {
			buf.Reset()
			logNotifier.Error(msg)
			output := buf.String()

			if !containsSubstring(output, "Hyprvoice Error") {
				t.Errorf("error log should contain 'Hyprvoice Error', got: %s", output)
			}

			if msg != "" && !containsSubstring(output, msg) {
				t.Errorf("error log should contain message %q, got: %s", msg, output)
			}
		}
	})
}

func TestNotifierMethods(t *testing.T) {
	// Test that all required methods exist and can be called
	var n Notifier

	// Test with each implementation
	implementations := []Notifier{
		Desktop{},
		Log{},
		Nop{},
	}

	for _, impl := range implementations {
		n = impl

		// Verify all methods exist by calling them
		n.RecordingStarted()
		n.RecordingEnded()
		n.Aborted()
		n.Transcribing()
		n.Error("test")
		n.Notify("test", "test")
	}
}

func TestNotifierEdgeCases(t *testing.T) {
	notifiers := []Notifier{Desktop{}, Log{}, Nop{}}

	t.Run("nil message handling", func(t *testing.T) {
		for _, notifier := range notifiers {
			// These should not panic even with empty strings
			notifier.Error("")
			notifier.Notify("", "")
			notifier.Notify("title", "")
			notifier.Notify("", "message")
		}
	})

	t.Run("concurrent access", func(t *testing.T) {
		for _, notifier := range notifiers {
			done := make(chan bool, 10)

			// Call methods concurrently
			for i := 0; i < 10; i++ {
				go func(id int) {
					notifier.RecordingStarted()
					notifier.RecordingEnded()
					notifier.Transcribing()
					notifier.Aborted()
					notifier.Error("concurrent test")
					notifier.Notify("title", "message")
					done <- true
				}(i)
			}

			// Wait for all goroutines
			for i := 0; i < 10; i++ {
				<-done
			}
		}
	})
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

// Simple substring search
func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
