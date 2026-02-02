package deps

import (
	"os/exec"
	"strings"
)

// Status represents the installation status of a dependency
type Status struct {
	Installed bool
	Path      string
	Version   string
}

// CheckWhisperCli checks if whisper-cli is installed and returns its status
func CheckWhisperCli() Status {
	path, err := exec.LookPath("whisper-cli")
	if err != nil {
		return Status{Installed: false}
	}

	status := Status{
		Installed: true,
		Path:      path,
	}

	// try to get version - whisper-cli --version outputs version info
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err == nil {
		// parse first line as version
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			status.Version = strings.TrimSpace(lines[0])
		}
	}

	return status
}

// CheckFFmpeg checks if ffmpeg is installed and returns its status
func CheckFFmpeg() Status {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return Status{Installed: false}
	}

	status := Status{
		Installed: true,
		Path:      path,
	}

	// ffmpeg -version outputs version info on first line
	cmd := exec.Command(path, "-version")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			status.Version = strings.TrimSpace(lines[0])
		}
	}

	return status
}
