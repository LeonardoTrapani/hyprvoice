package deps

import (
	"os/exec"
	"testing"
)

func TestCheckWhisperCli(t *testing.T) {
	status := CheckWhisperCli()

	// behavior depends on system - just verify no panic and correct structure
	if status.Installed {
		if status.Path == "" {
			t.Error("installed but path empty")
		}
	} else {
		if status.Path != "" {
			t.Error("not installed but path non-empty")
		}
	}
}

func TestCheckWhisperCli_NotInstalled(t *testing.T) {
	// if whisper-cli is not in PATH, should return Installed=false
	_, err := exec.LookPath("whisper-cli")
	if err != nil {
		status := CheckWhisperCli()
		if status.Installed {
			t.Error("expected Installed=false when whisper-cli not in PATH")
		}
		if status.Path != "" {
			t.Error("expected empty path when not installed")
		}
	} else {
		t.Skip("whisper-cli is installed, can't test not-installed case")
	}
}

func TestCheckFFmpeg(t *testing.T) {
	status := CheckFFmpeg()

	if status.Installed {
		if status.Path == "" {
			t.Error("installed but path empty")
		}
	} else {
		if status.Path != "" {
			t.Error("not installed but path non-empty")
		}
	}
}

func TestCheckFFmpeg_Installed(t *testing.T) {
	// ffmpeg is commonly installed - test if available
	_, err := exec.LookPath("ffmpeg")
	if err == nil {
		status := CheckFFmpeg()
		if !status.Installed {
			t.Error("ffmpeg in PATH but Installed=false")
		}
		if status.Path == "" {
			t.Error("ffmpeg installed but path empty")
		}
		// version should be populated
		if status.Version == "" {
			t.Error("ffmpeg installed but version empty")
		}
	} else {
		t.Skip("ffmpeg not installed, can't test installed case")
	}
}
