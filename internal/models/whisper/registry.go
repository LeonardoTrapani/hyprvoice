package whisper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// ProgressFunc is called during download with bytes downloaded and total
type ProgressFunc func(downloaded, total int64)

// IsInstalled returns true if the model is downloaded and available
func IsInstalled(modelID string) bool {
	path := GetModelPath(modelID)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

// ListInstalled returns IDs of all installed models
func ListInstalled() []string {
	var installed []string
	for _, m := range models {
		if IsInstalled(m.ID) {
			installed = append(installed, m.ID)
		}
	}
	return installed
}

// Download downloads a model from huggingface.
// Progress callback is optional (can be nil).
// Uses context for cancellation.
func Download(ctx context.Context, modelID string, onProgress ProgressFunc) error {
	info := GetModel(modelID)
	if info == nil {
		return fmt.Errorf("unknown model: %s", modelID)
	}

	url := GetDownloadURL(modelID)
	if url == "" {
		return fmt.Errorf("no download URL for model: %s", modelID)
	}

	// ensure directory exists
	dir, err := GetModelsDir()
	if err != nil {
		return fmt.Errorf("failed to get models directory: %w", err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	destPath := filepath.Join(dir, info.Filename)
	tempPath := destPath + ".downloading"

	// create temp file
	out, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		out.Close()
		os.Remove(tempPath) // clean up temp file on error
	}()

	// create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	total := resp.ContentLength
	if total < 0 {
		total = info.SizeBytes // fall back to expected size
	}

	var downloaded int64
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write: %w", writeErr)
			}
			downloaded += int64(n)
			if onProgress != nil {
				onProgress(downloaded, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read: %w", err)
		}
	}

	// close file before rename
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	// rename temp file to final destination
	if err := os.Rename(tempPath, destPath); err != nil {
		return fmt.Errorf("failed to finalize download: %w", err)
	}

	return nil
}

// Remove deletes a downloaded model
func Remove(modelID string) error {
	info := GetModel(modelID)
	if info == nil {
		return fmt.Errorf("unknown model: %s", modelID)
	}

	path := GetModelPath(modelID)
	if path == "" {
		return fmt.Errorf("failed to get model path")
	}

	if !IsInstalled(modelID) {
		return fmt.Errorf("model not installed: %s", modelID)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove model: %w", err)
	}

	return nil
}

// GetInstalledPath returns the path to an installed model, or error if not installed
func GetInstalledPath(modelID string) (string, error) {
	if !IsInstalled(modelID) {
		return "", fmt.Errorf("model not installed: %s", modelID)
	}
	return GetModelPath(modelID), nil
}
