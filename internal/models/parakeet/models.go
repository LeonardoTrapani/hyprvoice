package parakeet

import (
	"os"
	"path/filepath"
)

// ModelInfo holds metadata for a parakeet model
type ModelInfo struct {
	ID           string   // model identifier (e.g., "parakeet-tdt-0.6b-v3")
	Name         string   // display name (e.g., "Parakeet TDT 0.6B V3")
	Filename     string   // file name (e.g., "parakeet-tdt-0.6b-v3.q8_0.gguf")
	Size         string   // human readable size
	SizeBytes    int64    // size in bytes for progress tracking
	Languages    []string // supported language codes ("" = auto-detect)
	DocsURL      string   // model documentation URL
	DownloadURL  string   // full download URL
	NeedsRuntime bool     // requires nemo-speech runtime
}

// available parakeet models from huggingface.co/nvidia/parakeet-tdt-0.6b-v3
// https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3
const (
	baseDownloadURL = "https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3/resolve/main"
	docsURL         = "https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3"
)

// parakeet-tdt-0.6b-v3 supports 25 European languages with auto language detection
var europeanLanguages = []string{
	"en", "es", "fr", "de", "bg", "hr", "cs", "da", "nl", "et",
	"fi", "el", "hu", "it", "lv", "lt", "mt", "pl", "pt", "ro",
	"sk", "sl", "sv", "ru", "uk",
}

var models = []ModelInfo{
	{
		ID:           "parakeet-tdt-0.6b-v3",
		Name:         "Parakeet TDT 0.6B V3",
		Filename:     "parakeet-tdt-0.6b-v3.q8_0.gguf",
		Size:         "714MB",
		SizeBytes:    713_975_456,
		Languages:    europeanLanguages,
		DocsURL:      docsURL,
		DownloadURL:  baseDownloadURL + "/parakeet-tdt-0.6b-v3.q8_0.gguf",
		NeedsRuntime: true,
	},
}

// modelByID maps model ID to ModelInfo for quick lookup
var modelByID = func() map[string]ModelInfo {
	m := make(map[string]ModelInfo, len(models))
	for _, model := range models {
		m[model.ID] = model
	}
	return m
}()

// GetModelsDir returns the directory where parakeet models are stored.
// Creates the directory if it doesn't exist.
func GetModelsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "hyprvoice", "models", "parakeet")
	return dir, nil
}

// GetModelPath returns the full path to a model file.
// Returns empty string if model ID is unknown.
func GetModelPath(modelID string) string {
	info, ok := modelByID[modelID]
	if !ok {
		return ""
	}
	dir, err := GetModelsDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, info.Filename)
}

// GetModel returns info for a model by ID.
// Returns nil if model ID is unknown.
func GetModel(modelID string) *ModelInfo {
	info, ok := modelByID[modelID]
	if !ok {
		return nil
	}
	return &info
}

// ListModels returns all available parakeet models
func ListModels() []ModelInfo {
	result := make([]ModelInfo, len(models))
	copy(result, models)
	return result
}
