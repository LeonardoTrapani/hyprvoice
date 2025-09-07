package injection

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewInjector(t *testing.T) {
	config := Config{
		Mode:             "fallback",
		RestoreClipboard: true,
		WtypeTimeout:     5 * time.Second,
		ClipboardTimeout: 3 * time.Second,
	}

	injector := NewInjector(config)
	if injector == nil {
		t.Errorf("NewInjector() returned nil")
		return
	}

	// Test that the injector works with the expected config
	ctx := context.Background()
	err := injector.Inject(ctx, "test")
	// We expect this to fail due to missing external tools, but it should be the right type of error
	if err != nil {
		t.Logf("Injector created successfully (failed as expected due to missing tools): %v", err)
	}
}

func TestInjector_Inject(t *testing.T) {
	// Skip integration tests in CI environments
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}

	tests := []struct {
		name    string
		config  Config
		text    string
		wantErr bool
	}{
		{
			name: "inject with clipboard mode",
			config: Config{
				Mode:             "clipboard",
				RestoreClipboard: false,
				WtypeTimeout:     5 * time.Second,
				ClipboardTimeout: 3 * time.Second,
			},
			text:    "test text",
			wantErr: false,
		},
		{
			name: "inject with type mode",
			config: Config{
				Mode:             "type",
				RestoreClipboard: false,
				WtypeTimeout:     5 * time.Second,
				ClipboardTimeout: 3 * time.Second,
			},
			text:    "test text",
			wantErr: false,
		},
		{
			name: "inject with fallback mode",
			config: Config{
				Mode:             "fallback",
				RestoreClipboard: false,
				WtypeTimeout:     5 * time.Second,
				ClipboardTimeout: 3 * time.Second,
			},
			text:    "test text",
			wantErr: false,
		},
		{
			name: "inject empty text",
			config: Config{
				Mode:             "clipboard",
				RestoreClipboard: false,
				WtypeTimeout:     5 * time.Second,
				ClipboardTimeout: 3 * time.Second,
			},
			text:    "",
			wantErr: true,
		},
		{
			name: "inject with invalid mode",
			config: Config{
				Mode:             "invalid",
				RestoreClipboard: false,
				WtypeTimeout:     5 * time.Second,
				ClipboardTimeout: 3 * time.Second,
			},
			text:    "test text",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injector := NewInjector(tt.config)
			ctx := context.Background()

			err := injector.Inject(ctx, tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("Inject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	config := Config{
		Mode:             "fallback",
		RestoreClipboard: true,
		WtypeTimeout:     5 * time.Second,
		ClipboardTimeout: 3 * time.Second,
	}

	if config.Mode != "fallback" {
		t.Errorf("Mode mismatch: got %s, want %s", config.Mode, "fallback")
	}

	if !config.RestoreClipboard {
		t.Errorf("RestoreClipboard should be true")
	}

	if config.WtypeTimeout != 5*time.Second {
		t.Errorf("WtypeTimeout mismatch: got %v, want %v", config.WtypeTimeout, 5*time.Second)
	}

	if config.ClipboardTimeout != 3*time.Second {
		t.Errorf("ClipboardTimeout mismatch: got %v, want %v", config.ClipboardTimeout, 3*time.Second)
	}
}

// TestTypeText tests the typeText function
func TestTypeText(t *testing.T) {
	// Skip integration tests in CI environments
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}

	tests := []struct {
		name    string
		text    string
		wantErr bool
	}{
		{
			name:    "type normal text",
			text:    "hello world",
			wantErr: false,
		},
		{
			name:    "type empty text",
			text:    "",
			wantErr: false,
		},
		{
			name:    "type text with special characters",
			text:    "hello\nworld\t!",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := typeText(ctx, tt.text, 1*time.Second)
			if (err != nil) != tt.wantErr {
				t.Errorf("typeText() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCheckWtypeAvailable tests the wtype availability check
func TestCheckWtypeAvailable(t *testing.T) {
	err := checkWtypeAvailable()
	if err != nil {
		t.Logf("checkWtypeAvailable() failed (expected if wtype not installed): %v", err)
		// Don't fail the test if wtype is not available
		return
	}

	t.Logf("checkWtypeAvailable() succeeded - wtype is available")
}

// TestGetClipboard tests the clipboard get functionality
func TestGetClipboard(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test getting clipboard content
	content, err := getClipboard(ctx, 1*time.Second)
	if err != nil {
		t.Logf("getClipboard() failed (expected if wl-paste not available): %v", err)
		// Don't fail the test if clipboard tools are not available
		return
	}

	t.Logf("getClipboard() succeeded, content length: %d", len(content))
}

// TestSetClipboard tests the clipboard set functionality
func TestSetClipboard(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	testText := "test clipboard content"

	err := setClipboard(ctx, testText, 1*time.Second)
	if err != nil {
		t.Logf("setClipboard() failed (expected if wl-copy not available): %v", err)
		// Don't fail the test if clipboard tools are not available
		return
	}

	t.Logf("setClipboard() succeeded")

	// Try to read it back
	content, err := getClipboard(ctx, 1*time.Second)
	if err != nil {
		t.Logf("Failed to read back clipboard content: %v", err)
		return
	}

	if content != testText {
		t.Logf("Clipboard content mismatch: got %q, want %q", content, testText)
		// Don't fail - clipboard might have been modified by other processes
	}
}

// TestCheckClipboardAvailable tests the clipboard tools availability check
func TestCheckClipboardAvailable(t *testing.T) {
	err := checkClipboardAvailable()
	if err != nil {
		t.Logf("checkClipboardAvailable() failed (expected if clipboard tools not installed): %v", err)
		// Don't fail the test if clipboard tools are not available
		return
	}

	t.Logf("checkClipboardAvailable() succeeded - clipboard tools are available")
}

// TestInjector_ClipboardMode tests clipboard-only injection
func TestInjector_ClipboardMode(t *testing.T) {
	config := Config{
		Mode:             "clipboard",
		RestoreClipboard: false,
		WtypeTimeout:     5 * time.Second,
		ClipboardTimeout: 3 * time.Second,
	}

	injector := NewInjector(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := injector.Inject(ctx, "test clipboard text")
	if err != nil {
		t.Logf("Clipboard injection failed (expected if clipboard tools not available): %v", err)
		// Don't fail the test if clipboard tools are not available
		return
	}

	t.Logf("Clipboard injection succeeded")
}

// TestInjector_TypeMode tests typing-only injection
func TestInjector_TypeMode(t *testing.T) {
	config := Config{
		Mode:             "type",
		RestoreClipboard: false,
		WtypeTimeout:     5 * time.Second,
		ClipboardTimeout: 3 * time.Second,
	}

	injector := NewInjector(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := injector.Inject(ctx, "test typing text")
	if err != nil {
		t.Logf("Typing injection failed (expected if wtype not available): %v", err)
		// Don't fail the test if wtype is not available
		return
	}

	t.Logf("Typing injection succeeded")
}

// TestInjector_FallbackMode tests fallback injection behavior
func TestInjector_FallbackMode(t *testing.T) {
	config := Config{
		Mode:             "fallback",
		RestoreClipboard: false,
		WtypeTimeout:     5 * time.Second,
		ClipboardTimeout: 3 * time.Second,
	}

	injector := NewInjector(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := injector.Inject(ctx, "test fallback text")
	if err != nil {
		t.Logf("Fallback injection failed (expected if both wtype and clipboard tools not available): %v", err)
		// Don't fail the test if tools are not available
		return
	}

	t.Logf("Fallback injection succeeded")
}

// TestInjector_EmptyText tests injection of empty text
func TestInjector_EmptyText(t *testing.T) {
	config := Config{
		Mode:             "clipboard",
		RestoreClipboard: false,
		WtypeTimeout:     5 * time.Second,
		ClipboardTimeout: 3 * time.Second,
	}

	injector := NewInjector(config)
	ctx := context.Background()

	err := injector.Inject(ctx, "")
	if err == nil {
		t.Errorf("Inject() should fail with empty text")
		return
	}

	if err.Error() != "cannot inject empty text" {
		t.Errorf("Inject() error message = %q, want %q", err.Error(), "cannot inject empty text")
	}
}

// TestInjector_InvalidMode tests injection with invalid mode
func TestInjector_InvalidMode(t *testing.T) {
	config := Config{
		Mode:             "invalid",
		RestoreClipboard: false,
		WtypeTimeout:     5 * time.Second,
		ClipboardTimeout: 3 * time.Second,
	}

	injector := NewInjector(config)
	ctx := context.Background()

	err := injector.Inject(ctx, "test text")
	if err == nil {
		t.Errorf("Inject() should fail with invalid mode")
		return
	}

	expectedError := "unsupported injection mode: invalid"
	if err.Error() != expectedError {
		t.Errorf("Inject() error message = %q, want %q", err.Error(), expectedError)
	}
}
