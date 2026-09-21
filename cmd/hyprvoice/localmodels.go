package main

import (
	"context"
	"fmt"

	"github.com/leonardotrapani/hyprvoice/internal/models/parakeet"
	"github.com/leonardotrapani/hyprvoice/internal/models/whisper"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// localModelInstalled returns true if a local model has been downloaded.
// AdapterType determines which model registry (and download location) applies.
func localModelInstalled(m *provider.Model) bool {
	switch m.AdapterType {
	case provider.AdapterWhisperCpp:
		return whisper.IsInstalled(m.ID)
	case provider.AdapterParakeet:
		return parakeet.IsInstalled(m.ID)
	default:
		return false
	}
}

// localModelPath returns the full path of a downloaded local model.
func localModelPath(m *provider.Model) string {
	switch m.AdapterType {
	case provider.AdapterWhisperCpp:
		return whisper.GetModelPath(m.ID)
	case provider.AdapterParakeet:
		return parakeet.GetModelPath(m.ID)
	default:
		return ""
	}
}

// localModelDownload downloads a local model with progress reporting.
func localModelDownload(ctx context.Context, m *provider.Model, onProgress func(downloaded, total int64)) error {
	switch m.AdapterType {
	case provider.AdapterWhisperCpp:
		return whisper.Download(ctx, m.ID, onProgress)
	case provider.AdapterParakeet:
		return parakeet.Download(ctx, m.ID, onProgress)
	default:
		return fmt.Errorf("local download not supported for model: %s", m.ID)
	}
}

// localModelRemove removes a downloaded local model.
func localModelRemove(m *provider.Model) error {
	switch m.AdapterType {
	case provider.AdapterWhisperCpp:
		return whisper.Remove(m.ID)
	case provider.AdapterParakeet:
		return parakeet.Remove(m.ID)
	default:
		return fmt.Errorf("local removal not supported for model: %s", m.ID)
	}
}
