package whisper

import (
	"os"
	"path/filepath"
)

// ModelInfo holds metadata for a whisper model
type ModelInfo struct {
	ID           string // model identifier (e.g., "base.en")
	Name         string // display name (e.g., "Base English")
	Filename     string // file name (e.g., "ggml-base.en.bin")
	Size         string // human readable size
	SizeBytes    int64  // size in bytes for progress tracking
	Multilingual bool   // true if supports multiple languages
}

// available whisper models from huggingface.co/ggerganov/whisper.cpp
var models = []ModelInfo{
	// english-only models (faster, smaller)
	{ID: "tiny.en", Name: "Tiny English", Filename: "ggml-tiny.en.bin", Size: "75MB", SizeBytes: 75_000_000, Multilingual: false},
	{ID: "base.en", Name: "Base English", Filename: "ggml-base.en.bin", Size: "142MB", SizeBytes: 142_000_000, Multilingual: false},
	{ID: "small.en", Name: "Small English", Filename: "ggml-small.en.bin", Size: "466MB", SizeBytes: 466_000_000, Multilingual: false},
	{ID: "medium.en", Name: "Medium English", Filename: "ggml-medium.en.bin", Size: "1.5GB", SizeBytes: 1_500_000_000, Multilingual: false},

	// multilingual models
	{ID: "tiny", Name: "Tiny", Filename: "ggml-tiny.bin", Size: "75MB", SizeBytes: 75_000_000, Multilingual: true},
	{ID: "base", Name: "Base", Filename: "ggml-base.bin", Size: "142MB", SizeBytes: 142_000_000, Multilingual: true},
	{ID: "small", Name: "Small", Filename: "ggml-small.bin", Size: "466MB", SizeBytes: 466_000_000, Multilingual: true},
	{ID: "medium", Name: "Medium", Filename: "ggml-medium.bin", Size: "1.5GB", SizeBytes: 1_500_000_000, Multilingual: true},
	{ID: "large-v3", Name: "Large V3", Filename: "ggml-large-v3.bin", Size: "3GB", SizeBytes: 3_000_000_000, Multilingual: true},
}

// modelByID maps model ID to ModelInfo for quick lookup
var modelByID = func() map[string]ModelInfo {
	m := make(map[string]ModelInfo, len(models))
	for _, model := range models {
		m[model.ID] = model
	}
	return m
}()

const (
	// base URL for downloading models from huggingface
	baseDownloadURL = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main"
)

// GetModelsDir returns the directory where whisper models are stored.
// Creates the directory if it doesn't exist.
func GetModelsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "hyprvoice", "models", "whisper")
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

// GetDownloadURL returns the full download URL for a model.
// Returns empty string if model ID is unknown.
func GetDownloadURL(modelID string) string {
	info, ok := modelByID[modelID]
	if !ok {
		return ""
	}
	return baseDownloadURL + "/" + info.Filename
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

// ListModels returns all available whisper models
func ListModels() []ModelInfo {
	result := make([]ModelInfo, len(models))
	copy(result, models)
	return result
}

// ListMultilingualModels returns models that support multiple languages
func ListMultilingualModels() []ModelInfo {
	var result []ModelInfo
	for _, m := range models {
		if m.Multilingual {
			result = append(result, m)
		}
	}
	return result
}

// ListEnglishOnlyModels returns english-only models
func ListEnglishOnlyModels() []ModelInfo {
	var result []ModelInfo
	for _, m := range models {
		if !m.Multilingual {
			result = append(result, m)
		}
	}
	return result
}
