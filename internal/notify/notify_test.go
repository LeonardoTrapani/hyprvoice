package notify

import (
	"testing"

	"github.com/leonardotrapani/hyprvoice/internal/config"
)

func TestDesktop_Notify(t *testing.T) {
	desktop := Desktop{}

	// Test normal notification
	desktop.Notify("Test Title", "Test Message")

	// Test error notification
	desktop.Error("Test Error Message")

	// Test specific methods
	desktop.RecordingStarted()
	desktop.Transcribing()
}

func TestLog_Notify(t *testing.T) {
	logNotifier := Log{}

	// Test normal notification
	logNotifier.Notify("Test Title", "Test Message")

	// Test error notification
	logNotifier.Error("Test Error Message")
}

func TestNop_Notify(t *testing.T) {
	nop := Nop{}

	// Test that these methods don't panic
	nop.Notify("Test Title", "Test Message")
	nop.Error("Test Error Message")
}

func TestGetNotifierBasedOnConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected string
	}{
		{
			name: "desktop notification type",
			config: &config.Config{
				Notifications: config.NotificationsConfig{
					Type: "desktop",
				},
			},
			expected: "desktop",
		},
		{
			name: "log notification type",
			config: &config.Config{
				Notifications: config.NotificationsConfig{
					Type: "log",
				},
			},
			expected: "log",
		},
		{
			name: "none notification type",
			config: &config.Config{
				Notifications: config.NotificationsConfig{
					Type: "none",
				},
			},
			expected: "nop",
		},
		{
			name: "unknown notification type",
			config: &config.Config{
				Notifications: config.NotificationsConfig{
					Type: "unknown",
				},
			},
			expected: "nop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := GetNotifierBasedOnConfig(tt.config)

			// Test the notifier by calling its methods
			notifier.Notify("Test", "Message")
			notifier.Error("Error")

			// Check the type by testing behavior
			switch tt.expected {
			case "desktop":
				// Desktop notifier should not panic
				if desktop, ok := notifier.(Desktop); ok {
					desktop.RecordingStarted()
					desktop.Transcribing()
				}
			case "log":
				// Log notifier should not panic
				if logNotifier, ok := notifier.(Log); ok {
					logNotifier.Notify("Test", "Message")
					logNotifier.Error("Error")
				}
			case "nop":
				// Nop notifier should not panic
				if nop, ok := notifier.(Nop); ok {
					nop.Notify("Test", "Message")
					nop.Error("Error")
				}
			}
		})
	}
}

func TestDesktop_Methods(t *testing.T) {
	desktop := Desktop{}

	// Test RecordingStarted method
	desktop.RecordingStarted()

	// Test Transcribing method
	desktop.Transcribing()

	// Test Error method
	desktop.Error("Test error")

	// Test Notify method
	desktop.Notify("Test Title", "Test Message")
}

func TestLog_Methods(t *testing.T) {
	logNotifier := Log{}

	// Test Error method
	logNotifier.Error("Test error")

	// Test Notify method
	logNotifier.Notify("Test Title", "Test Message")
}

func TestNop_Methods(t *testing.T) {
	nop := Nop{}

	// Test Error method
	nop.Error("Test error")

	// Test Notify method
	nop.Notify("Test Title", "Test Message")
}

func TestNotifierInterface(t *testing.T) {
	// Test that all notifiers implement the Notifier interface
	var notifier Notifier

	// Test Desktop
	notifier = Desktop{}
	notifier.Notify("Test", "Message")
	notifier.Error("Error")

	// Test Log
	notifier = Log{}
	notifier.Notify("Test", "Message")
	notifier.Error("Error")

	// Test Nop
	notifier = Nop{}
	notifier.Notify("Test", "Message")
	notifier.Error("Error")
}

func TestNotificationTypes(t *testing.T) {
	// Test different notification configurations
	configs := []*config.Config{
		{
			Notifications: config.NotificationsConfig{
				Type: "desktop",
			},
		},
		{
			Notifications: config.NotificationsConfig{
				Type: "log",
			},
		},
		{
			Notifications: config.NotificationsConfig{
				Type: "none",
			},
		},
	}

	for _, cfg := range configs {
		t.Run("type_"+cfg.Notifications.Type, func(t *testing.T) {
			notifier := GetNotifierBasedOnConfig(cfg)

			// Test that the notifier works
			notifier.Notify("Test Title", "Test Message")
			notifier.Error("Test Error")
		})
	}
}
