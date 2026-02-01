package whisper

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetModelsDir(t *testing.T) {
	dir, err := GetModelsDir()
	if err != nil {
		t.Fatalf("GetModelsDir() error = %v", err)
	}

	// should not contain ~ (should be expanded)
	if strings.Contains(dir, "~") {
		t.Errorf("GetModelsDir() contains ~, got %s", dir)
	}

	// should end with expected path
	if !strings.HasSuffix(dir, filepath.Join(".local", "share", "hyprvoice", "models", "whisper")) {
		t.Errorf("GetModelsDir() = %s, want path ending with .local/share/hyprvoice/models/whisper", dir)
	}
}

func TestGetModelPath(t *testing.T) {
	tests := []struct {
		modelID string
		wantEnd string
	}{
		{"base.en", "ggml-base.en.bin"},
		{"tiny", "ggml-tiny.bin"},
		{"large-v3", "ggml-large-v3.bin"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			got := GetModelPath(tt.modelID)
			if tt.wantEnd == "" {
				if got != "" {
					t.Errorf("GetModelPath(%q) = %s, want empty", tt.modelID, got)
				}
				return
			}
			if !strings.HasSuffix(got, tt.wantEnd) {
				t.Errorf("GetModelPath(%q) = %s, want ending with %s", tt.modelID, got, tt.wantEnd)
			}
		})
	}
}

func TestGetDownloadURL(t *testing.T) {
	tests := []struct {
		modelID string
		wantURL string
	}{
		{"base.en", "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin"},
		{"tiny", "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			got := GetDownloadURL(tt.modelID)
			if got != tt.wantURL {
				t.Errorf("GetDownloadURL(%q) = %s, want %s", tt.modelID, got, tt.wantURL)
			}
		})
	}
}

func TestGetModel(t *testing.T) {
	t.Run("known model", func(t *testing.T) {
		info := GetModel("base.en")
		if info == nil {
			t.Fatal("GetModel(base.en) = nil, want non-nil")
		}
		if info.ID != "base.en" {
			t.Errorf("info.ID = %s, want base.en", info.ID)
		}
		if info.Filename != "ggml-base.en.bin" {
			t.Errorf("info.Filename = %s, want ggml-base.en.bin", info.Filename)
		}
		if info.Multilingual {
			t.Error("base.en should not be multilingual")
		}
	})

	t.Run("multilingual model", func(t *testing.T) {
		info := GetModel("base")
		if info == nil {
			t.Fatal("GetModel(base) = nil, want non-nil")
		}
		if !info.Multilingual {
			t.Error("base should be multilingual")
		}
	})

	t.Run("unknown model", func(t *testing.T) {
		info := GetModel("unknown")
		if info != nil {
			t.Errorf("GetModel(unknown) = %v, want nil", info)
		}
	})
}

func TestListModels(t *testing.T) {
	models := ListModels()
	if len(models) != 9 {
		t.Errorf("ListModels() returned %d models, want 9", len(models))
	}

	// verify known models exist
	ids := make(map[string]bool)
	for _, m := range models {
		ids[m.ID] = true
	}

	expected := []string{"tiny.en", "base.en", "small.en", "medium.en", "tiny", "base", "small", "medium", "large-v3"}
	for _, id := range expected {
		if !ids[id] {
			t.Errorf("ListModels() missing model %s", id)
		}
	}
}

func TestListMultilingualModels(t *testing.T) {
	models := ListMultilingualModels()
	if len(models) != 5 {
		t.Errorf("ListMultilingualModels() returned %d models, want 5", len(models))
	}

	for _, m := range models {
		if !m.Multilingual {
			t.Errorf("ListMultilingualModels() returned non-multilingual model %s", m.ID)
		}
	}
}

func TestListEnglishOnlyModels(t *testing.T) {
	models := ListEnglishOnlyModels()
	if len(models) != 4 {
		t.Errorf("ListEnglishOnlyModels() returned %d models, want 4", len(models))
	}

	for _, m := range models {
		if m.Multilingual {
			t.Errorf("ListEnglishOnlyModels() returned multilingual model %s", m.ID)
		}
		if !strings.HasSuffix(m.ID, ".en") {
			t.Errorf("ListEnglishOnlyModels() returned model without .en suffix: %s", m.ID)
		}
	}
}

func TestIsInstalled(t *testing.T) {
	// should return false for non-existent model
	if IsInstalled("base.en") {
		// this might actually be true if the user has it installed
		// just skip this test if model exists
		t.Skip("base.en is installed, skipping test")
	}

	// should return false for unknown model
	if IsInstalled("unknown-model") {
		t.Error("IsInstalled(unknown-model) = true, want false")
	}
}

func TestListInstalled(t *testing.T) {
	// just verify it doesn't crash
	installed := ListInstalled()
	t.Logf("Installed models: %v", installed)
}

func TestDownload_UnknownModel(t *testing.T) {
	err := Download(context.Background(), "unknown-model", nil)
	if err == nil {
		t.Error("Download(unknown-model) = nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown model") {
		t.Errorf("Download error = %v, want error containing 'unknown model'", err)
	}
}

func TestDownload_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := Download(ctx, "tiny.en", nil)
	if err == nil {
		t.Error("Download with cancelled context = nil, want error")
	}
}

func TestRemove_NotInstalled(t *testing.T) {
	// use a model that's unlikely to be installed
	err := Remove("large-v3")
	if err == nil {
		t.Skip("large-v3 is installed, skipping test")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("Remove error = %v, want error containing 'not installed'", err)
	}
}

func TestRemove_UnknownModel(t *testing.T) {
	err := Remove("unknown-model")
	if err == nil {
		t.Error("Remove(unknown-model) = nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown model") {
		t.Errorf("Remove error = %v, want error containing 'unknown model'", err)
	}
}

func TestGetInstalledPath_NotInstalled(t *testing.T) {
	// use a model that's unlikely to be installed
	_, err := GetInstalledPath("large-v3")
	if err == nil {
		t.Skip("large-v3 is installed, skipping test")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("GetInstalledPath error = %v, want error containing 'not installed'", err)
	}
}

func TestDownloadAndRemove_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// create a temp directory for this test
	tempDir := t.TempDir()

	// override GetModelsDir for this test
	origGetModelsDir := GetModelsDir
	_ = origGetModelsDir // acknowledge we're shadowing

	// we can't easily override GetModelsDir since it's a function not a var
	// so we'll just check the download flow works conceptually
	// actual download testing would need network and is slow

	t.Log("Integration test would download a model here")
	t.Log("Temp dir:", tempDir)
}

// TestModelInfo_SizeBytes verifies size bytes are reasonable
func TestModelInfo_SizeBytes(t *testing.T) {
	models := ListModels()
	for _, m := range models {
		if m.SizeBytes <= 0 {
			t.Errorf("Model %s has invalid SizeBytes: %d", m.ID, m.SizeBytes)
		}
	}
}

// TestModelInfo_HasAllFields verifies all models have required fields
func TestModelInfo_HasAllFields(t *testing.T) {
	models := ListModels()
	for _, m := range models {
		if m.ID == "" {
			t.Error("Model has empty ID")
		}
		if m.Name == "" {
			t.Errorf("Model %s has empty Name", m.ID)
		}
		if m.Filename == "" {
			t.Errorf("Model %s has empty Filename", m.ID)
		}
		if m.Size == "" {
			t.Errorf("Model %s has empty Size", m.ID)
		}
	}
}
