package bus

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPidManager_CheckExisting(t *testing.T) {
	// Test with no existing PID file
	t.Run("no existing PID file", func(t *testing.T) {
		tempDir := t.TempDir()
		originalCacheDir := os.Getenv("XDG_CACHE_HOME")
		os.Setenv("XDG_CACHE_HOME", tempDir)
		defer func() {
			if originalCacheDir == "" {
				os.Unsetenv("XDG_CACHE_HOME")
			} else {
				os.Setenv("XDG_CACHE_HOME", originalCacheDir)
			}
		}()

		pm, err := newPidManager()
		if err != nil {
			t.Fatalf("Failed to create PID manager: %v", err)
		}

		err = pm.checkExisting()
		if err != nil {
			t.Errorf("checkExisting() error = %v, want no error", err)
		}
	})

	// Test with invalid PID in file
	t.Run("invalid PID in file", func(t *testing.T) {
		tempDir := t.TempDir()
		originalCacheDir := os.Getenv("XDG_CACHE_HOME")
		os.Setenv("XDG_CACHE_HOME", tempDir)
		defer func() {
			if originalCacheDir == "" {
				os.Unsetenv("XDG_CACHE_HOME")
			} else {
				os.Setenv("XDG_CACHE_HOME", originalCacheDir)
			}
		}()

		// Create PID file with invalid content
		pidPath := filepath.Join(tempDir, "hyprvoice", PidName)
		os.MkdirAll(filepath.Dir(pidPath), 0755)
		err := os.WriteFile(pidPath, []byte("invalid"), 0644)
		if err != nil {
			t.Fatalf("Failed to create PID file: %v", err)
		}

		pm, err := newPidManager()
		if err != nil {
			t.Fatalf("Failed to create PID manager: %v", err)
		}

		err = pm.checkExisting()
		if err != nil {
			t.Errorf("checkExisting() error = %v, want no error", err)
		}
	})
}

func TestPidManager_Create(t *testing.T) {
	tempDir := t.TempDir()
	originalCacheDir := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer func() {
		if originalCacheDir == "" {
			os.Unsetenv("XDG_CACHE_HOME")
		} else {
			os.Setenv("XDG_CACHE_HOME", originalCacheDir)
		}
	}()

	pm, err := newPidManager()
	if err != nil {
		t.Fatalf("Failed to create PID manager: %v", err)
	}

	err = pm.create()
	if err != nil {
		t.Errorf("create() error = %v", err)
		return
	}

	// Verify PID file was created
	pidPath := filepath.Join(tempDir, "hyprvoice", PidName)
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		t.Errorf("create() did not create PID file")
		return
	}
}

func TestPidManager_Remove(t *testing.T) {
	tempDir := t.TempDir()
	originalCacheDir := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer func() {
		if originalCacheDir == "" {
			os.Unsetenv("XDG_CACHE_HOME")
		} else {
			os.Setenv("XDG_CACHE_HOME", originalCacheDir)
		}
	}()

	// Create a PID file first
	pidPath := filepath.Join(tempDir, "hyprvoice", PidName)
	os.MkdirAll(filepath.Dir(pidPath), 0755)
	err := os.WriteFile(pidPath, []byte("1234"), 0644)
	if err != nil {
		t.Fatalf("Failed to create PID file: %v", err)
	}

	pm, err := newPidManager()
	if err != nil {
		t.Fatalf("Failed to create PID manager: %v", err)
	}

	err = pm.remove()
	if err != nil {
		t.Errorf("remove() error = %v", err)
		return
	}

	// Verify PID file was removed
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Errorf("remove() did not remove PID file")
	}
}

func TestSocketManager_Listen(t *testing.T) {
	tempDir := t.TempDir()
	originalCacheDir := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer func() {
		if originalCacheDir == "" {
			os.Unsetenv("XDG_CACHE_HOME")
		} else {
			os.Setenv("XDG_CACHE_HOME", originalCacheDir)
		}
	}()

	sm, err := newSocketManager()
	if err != nil {
		t.Fatalf("Failed to create socket manager: %v", err)
	}

	listener, err := sm.listen()
	if err != nil {
		t.Errorf("listen() error = %v", err)
		return
	}
	defer listener.Close()

	// Verify socket file was created
	sockPath := filepath.Join(tempDir, "hyprvoice", SockName)
	if _, err := os.Stat(sockPath); os.IsNotExist(err) {
		t.Errorf("listen() did not create socket file")
	}
}

func TestSocketManager_Dial(t *testing.T) {
	tempDir := t.TempDir()
	originalCacheDir := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer func() {
		if originalCacheDir == "" {
			os.Unsetenv("XDG_CACHE_HOME")
		} else {
			os.Setenv("XDG_CACHE_HOME", originalCacheDir)
		}
	}()

	// Start a test server
	sm, err := newSocketManager()
	if err != nil {
		t.Fatalf("Failed to create socket manager: %v", err)
	}

	listener, err := sm.listen()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	// Give the server a moment to start
	time.Sleep(10 * time.Millisecond)

	// Test dialing
	conn, err := sm.dial()
	if err != nil {
		t.Errorf("dial() error = %v", err)
		return
	}
	defer conn.Close()

	// Verify connection is working
	if conn == nil {
		t.Errorf("dial() returned nil connection")
	}
}

func TestSendCommand(t *testing.T) {
	tempDir := t.TempDir()
	originalCacheDir := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer func() {
		if originalCacheDir == "" {
			os.Unsetenv("XDG_CACHE_HOME")
		} else {
			os.Setenv("XDG_CACHE_HOME", originalCacheDir)
		}
	}()

	// Start a test server
	sm, err := newSocketManager()
	if err != nil {
		t.Fatalf("Failed to create socket manager: %v", err)
	}

	listener, err := sm.listen()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	// Handle test commands
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			// Read command
			buf := make([]byte, 1)
			n, err := conn.Read(buf)
			if err != nil || n != 1 {
				conn.Close()
				continue
			}

			// Respond based on command
			var response string
			switch buf[0] {
			case 't':
				response = "OK toggled\n"
			case 's':
				response = "STATUS status=idle\n"
			case 'v':
				response = "STATUS proto=0.1\n"
			case 'q':
				response = "OK quitting\n"
			default:
				response = "ERR unknown\n"
			}

			conn.Write([]byte(response))
			conn.Close()
		}
	}()

	// Give the server a moment to start
	time.Sleep(10 * time.Millisecond)

	// Test toggle command
	response, err := SendCommand('t')
	if err != nil {
		t.Errorf("SendCommand() error = %v", err)
		return
	}

	if response != "OK toggled\n" {
		t.Errorf("SendCommand() = %q, want %q", response, "OK toggled\n")
	}
}

func TestCheckExistingDaemon(t *testing.T) {
	// Test with no existing daemon
	t.Run("no existing daemon", func(t *testing.T) {
		tempDir := t.TempDir()
		originalCacheDir := os.Getenv("XDG_CACHE_HOME")
		os.Setenv("XDG_CACHE_HOME", tempDir)
		defer func() {
			if originalCacheDir == "" {
				os.Unsetenv("XDG_CACHE_HOME")
			} else {
				os.Setenv("XDG_CACHE_HOME", originalCacheDir)
			}
		}()

		err := CheckExistingDaemon()
		if err != nil {
			t.Errorf("CheckExistingDaemon() error = %v, want no error", err)
		}
	})
}

func TestCreatePidFile(t *testing.T) {
	tempDir := t.TempDir()
	originalCacheDir := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer func() {
		if originalCacheDir == "" {
			os.Unsetenv("XDG_CACHE_HOME")
		} else {
			os.Setenv("XDG_CACHE_HOME", originalCacheDir)
		}
	}()

	err := CreatePidFile()
	if err != nil {
		t.Errorf("CreatePidFile() error = %v", err)
		return
	}

	// Verify PID file was created
	pidPath := filepath.Join(tempDir, "hyprvoice", PidName)
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		t.Errorf("CreatePidFile() did not create PID file")
	}
}

func TestRemovePidFile(t *testing.T) {
	tempDir := t.TempDir()
	originalCacheDir := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer func() {
		if originalCacheDir == "" {
			os.Unsetenv("XDG_CACHE_HOME")
		} else {
			os.Setenv("XDG_CACHE_HOME", originalCacheDir)
		}
	}()

	// Create a PID file first
	pidPath := filepath.Join(tempDir, "hyprvoice", PidName)
	os.MkdirAll(filepath.Dir(pidPath), 0755)
	err := os.WriteFile(pidPath, []byte("1234"), 0644)
	if err != nil {
		t.Fatalf("Failed to create PID file: %v", err)
	}

	err = RemovePidFile()
	if err != nil {
		t.Errorf("RemovePidFile() error = %v", err)
		return
	}

	// Verify PID file was removed
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Errorf("RemovePidFile() did not remove PID file")
	}
}

func TestListen(t *testing.T) {
	tempDir := t.TempDir()
	originalCacheDir := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer func() {
		if originalCacheDir == "" {
			os.Unsetenv("XDG_CACHE_HOME")
		} else {
			os.Setenv("XDG_CACHE_HOME", originalCacheDir)
		}
	}()

	listener, err := Listen()
	if err != nil {
		t.Errorf("Listen() error = %v", err)
		return
	}
	defer listener.Close()

	// Verify socket file was created
	sockPath := filepath.Join(tempDir, "hyprvoice", SockName)
	if _, err := os.Stat(sockPath); os.IsNotExist(err) {
		t.Errorf("Listen() did not create socket file")
	}
}

func TestDial(t *testing.T) {
	tempDir := t.TempDir()
	originalCacheDir := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer func() {
		if originalCacheDir == "" {
			os.Unsetenv("XDG_CACHE_HOME")
		} else {
			os.Setenv("XDG_CACHE_HOME", originalCacheDir)
		}
	}()

	// Start a test server
	listener, err := Listen()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	// Give the server a moment to start
	time.Sleep(10 * time.Millisecond)

	// Test dialing
	conn, err := Dial()
	if err != nil {
		t.Errorf("Dial() error = %v", err)
		return
	}
	defer conn.Close()

	// Verify connection is working
	if conn == nil {
		t.Errorf("Dial() returned nil connection")
	}
}
