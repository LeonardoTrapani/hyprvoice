package injection

import (
	"context"
	"fmt"
	"time"
)

type Injector interface {
	Inject(ctx context.Context, text string) error
}

type Config struct {
	Mode             string        // "clipboard", "type", "fallback"
	RestoreClipboard bool          // Restore original clipboard after injection
	WtypeTimeout     time.Duration // Timeout for wtype commands
	ClipboardTimeout time.Duration // Timeout for clipboard operations
}

type injector struct {
	config Config
}

func NewInjector(config Config) Injector {
	return &injector{
		config: config,
	}
}

func (i *injector) Inject(ctx context.Context, text string) error {
	if text == "" {
		return fmt.Errorf("cannot inject empty text")
	}

	// Copy to clipboard for clipboard mode and fallback mode
	var originalClipboard string
	var err error

	if i.config.Mode == "clipboard" || i.config.Mode == "fallback" {
		if err := checkClipboardAvailable(); err != nil {
			return fmt.Errorf("clipboard tools not available: %w", err)
		}

		if i.config.RestoreClipboard {
			originalClipboard, _ = getClipboard(ctx, i.config.ClipboardTimeout)
		}

		if err := setClipboard(ctx, text, i.config.ClipboardTimeout); err != nil {
			return fmt.Errorf("failed to copy text to clipboard: %w", err)
		}
	}

	// Handle different injection modes
	switch i.config.Mode {
	case "clipboard":
		// Already handled above
		return nil

	case "type":
		err = typeText(ctx, text, i.config.WtypeTimeout)
		if err != nil {
			return fmt.Errorf("failed to type text: %w", err)
		}

	case "fallback":
		// Try typing first, fallback to clipboard
		err = typeText(ctx, text, i.config.WtypeTimeout)
		if err != nil {
			// Typing failed, but clipboard is already set from above
			// Just log the typing error but don't fail the injection
			return nil
		}

	default:
		return fmt.Errorf("unsupported injection mode: %s", i.config.Mode)
	}

	// Restore original clipboard if configured and we have it
	if i.config.RestoreClipboard && originalClipboard != "" {
		go func() {
			time.Sleep(100 * time.Millisecond)
			restoreCtx, cancel := context.WithTimeout(ctx, i.config.ClipboardTimeout)
			defer cancel()
			setClipboard(restoreCtx, originalClipboard, i.config.ClipboardTimeout)
		}()
	}

	return nil
}
