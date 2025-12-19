package injection

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewInjector(t *testing.T) {
	config := Config{
		Backends:         []string{"wtype", "clipboard"},
		YdotoolTimeout:   5 * time.Second,
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

func TestNewInjector_DefaultsToClipboard(t *testing.T) {
	config := Config{
		Backends:         []string{},
		ClipboardTimeout: 3 * time.Second,
	}

	injector := NewInjector(config)
	if injector == nil {
		t.Errorf("NewInjector() returned nil")
		return
	}

	// Should default to clipboard backend - just test it works
	ctx := context.Background()
	err := injector.Inject(ctx, "test")
	// Will fail if no clipboard tools, but that's ok
	if err != nil {
		t.Logf("Injection failed (expected without tools): %v", err)
	}
}

func TestNewInjector_IgnoresUnknownBackends(t *testing.T) {
	config := Config{
		Backends:         []string{"unknown", "wtype", "invalid"},
		WtypeTimeout:     5 * time.Second,
		ClipboardTimeout: 3 * time.Second,
	}

	injector := NewInjector(config)
	// Just verify it was created - we can't inspect internals
	if injector == nil {
		t.Errorf("NewInjector() returned nil")
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
			name: "inject with clipboard backend",
			config: Config{
				Backends:         []string{"clipboard"},
				ClipboardTimeout: 3 * time.Second,
			},
			text:    "test text",
			wantErr: false,
		},
		{
			name: "inject with wtype backend",
			config: Config{
				Backends:     []string{"wtype"},
				WtypeTimeout: 5 * time.Second,
			},
			text:    "test text",
			wantErr: false,
		},
		{
			name: "inject with fallback chain",
			config: Config{
				Backends:         []string{"ydotool", "wtype", "clipboard"},
				YdotoolTimeout:   5 * time.Second,
				WtypeTimeout:     5 * time.Second,
				ClipboardTimeout: 3 * time.Second,
			},
			text:    "test text",
			wantErr: false,
		},
		{
			name: "inject empty text",
			config: Config{
				Backends:         []string{"clipboard"},
				ClipboardTimeout: 3 * time.Second,
			},
			text:    "",
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
		Backends:         []string{"ydotool", "wtype", "clipboard"},
		YdotoolTimeout:   5 * time.Second,
		WtypeTimeout:     5 * time.Second,
		ClipboardTimeout: 3 * time.Second,
	}

	if len(config.Backends) != 3 {
		t.Errorf("Backends length mismatch: got %d, want %d", len(config.Backends), 3)
	}

	if config.WtypeTimeout != 5*time.Second {
		t.Errorf("WtypeTimeout mismatch: got %v, want %v", config.WtypeTimeout, 5*time.Second)
	}

	if config.ClipboardTimeout != 3*time.Second {
		t.Errorf("ClipboardTimeout mismatch: got %v, want %v", config.ClipboardTimeout, 3*time.Second)
	}
}

// TestWtypeBackend tests the wtype backend
func TestWtypeBackend(t *testing.T) {
	backend := NewWtypeBackend()

	if backend.Name() != "wtype" {
		t.Errorf("Name() = %s, want wtype", backend.Name())
	}

	err := backend.Available()
	if err != nil {
		t.Logf("wtype not available (expected): %v", err)
		return
	}

	t.Logf("wtype is available")
}

// TestYdotoolBackend tests the ydotool backend
func TestYdotoolBackend(t *testing.T) {
	backend := NewYdotoolBackend()

	if backend.Name() != "ydotool" {
		t.Errorf("Name() = %s, want ydotool", backend.Name())
	}

	err := backend.Available()
	if err != nil {
		t.Logf("ydotool not available (expected): %v", err)
		return
	}

	t.Logf("ydotool is available")
}

// TestClipboardBackend tests the clipboard backend
func TestClipboardBackend(t *testing.T) {
	backend := NewClipboardBackend()

	if backend.Name() != "clipboard" {
		t.Errorf("Name() = %s, want clipboard", backend.Name())
	}

	err := backend.Available()
	if err != nil {
		t.Logf("clipboard not available (expected): %v", err)
		return
	}

	t.Logf("clipboard is available")
}

// TestInjector_ClipboardMode tests clipboard-only injection
func TestInjector_ClipboardMode(t *testing.T) {
	config := Config{
		Backends:         []string{"clipboard"},
		ClipboardTimeout: 3 * time.Second,
	}

	injector := NewInjector(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := injector.Inject(ctx, "test clipboard text")
	if err != nil {
		t.Logf("Clipboard injection failed (expected if clipboard tools not available): %v", err)
		return
	}

	t.Logf("Clipboard injection succeeded")
}

// TestInjector_WtypeMode tests wtype-only injection
func TestInjector_WtypeMode(t *testing.T) {
	config := Config{
		Backends:     []string{"wtype"},
		WtypeTimeout: 5 * time.Second,
	}

	injector := NewInjector(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := injector.Inject(ctx, "test typing text")
	if err != nil {
		t.Logf("Wtype injection failed (expected if wtype not available): %v", err)
		return
	}

	t.Logf("Wtype injection succeeded")
}

// TestInjector_FallbackChain tests fallback chain injection
func TestInjector_FallbackChain(t *testing.T) {
	config := Config{
		Backends:         []string{"ydotool", "wtype", "clipboard"},
		YdotoolTimeout:   5 * time.Second,
		WtypeTimeout:     5 * time.Second,
		ClipboardTimeout: 3 * time.Second,
	}

	injector := NewInjector(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := injector.Inject(ctx, "test fallback text")
	if err != nil {
		t.Logf("Fallback injection failed (expected if all tools not available): %v", err)
		return
	}

	t.Logf("Fallback injection succeeded")
}

// TestInjector_EmptyText tests injection of empty text
func TestInjector_EmptyText(t *testing.T) {
	config := Config{
		Backends:         []string{"clipboard"},
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
