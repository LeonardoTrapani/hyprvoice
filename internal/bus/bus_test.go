package bus

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestPidManagerBasics(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a custom pidManager for testing
	testPidManager := &pidManager{
		path: filepath.Join(tempDir, PidName),
	}

	t.Run("create and remove PID file", func(t *testing.T) {
		// Create PID file
		err := testPidManager.create()
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}

		// Check file exists and contains current PID
		pidData, err := os.ReadFile(testPidManager.path)
		if err != nil {
			t.Fatalf("failed to read PID file: %v", err)
		}

		expectedPid := strconv.Itoa(os.Getpid())
		if string(pidData) != expectedPid {
			t.Errorf("PID file contains %q, expected %q", string(pidData), expectedPid)
		}

		// Remove PID file
		err = testPidManager.remove()
		if err != nil {
			t.Fatalf("remove failed: %v", err)
		}

		// Check file no longer exists
		if _, err := os.Stat(testPidManager.path); !os.IsNotExist(err) {
			t.Error("PID file should not exist after removal")
		}
	})

	t.Run("checkExisting with no PID file", func(t *testing.T) {
		err := testPidManager.checkExisting()
		if err != nil {
			t.Errorf("checkExisting should not error when no PID file exists: %v", err)
		}
	})

	t.Run("checkExisting with current process", func(t *testing.T) {
		// Create PID file with current process
		err := testPidManager.create()
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}
		defer testPidManager.remove()

		// Check should fail because process is running
		err = testPidManager.checkExisting()
		if err == nil {
			t.Error("checkExisting should fail when process is running")
		}
	})

	t.Run("checkExisting with stale PID file", func(t *testing.T) {
		// Create PID file with non-existent PID
		stalePid := "99999"
		err := os.WriteFile(testPidManager.path, []byte(stalePid), 0o600)
		if err != nil {
			t.Fatalf("failed to write stale PID file: %v", err)
		}

		// Check should succeed and remove stale file
		err = testPidManager.checkExisting()
		if err != nil {
			t.Errorf("checkExisting should succeed with stale PID: %v", err)
		}

		// File should be removed
		if _, err := os.Stat(testPidManager.path); !os.IsNotExist(err) {
			t.Error("stale PID file should be removed")
		}
	})

	t.Run("checkExisting with invalid PID file", func(t *testing.T) {
		// Create PID file with invalid content
		err := os.WriteFile(testPidManager.path, []byte("invalid"), 0o600)
		if err != nil {
			t.Fatalf("failed to write invalid PID file: %v", err)
		}

		// Check should succeed and remove invalid file
		err = testPidManager.checkExisting()
		if err != nil {
			t.Errorf("checkExisting should succeed with invalid PID: %v", err)
		}

		// File should be removed
		if _, err := os.Stat(testPidManager.path); !os.IsNotExist(err) {
			t.Error("invalid PID file should be removed")
		}
	})
}

func TestIsProcessAlive(t *testing.T) {
	pm := &pidManager{}

	t.Run("current process", func(t *testing.T) {
		if !pm.isProcessAlive(os.Getpid()) {
			t.Error("current process should be alive")
		}
	})

	t.Run("non-existent process", func(t *testing.T) {
		// Use a PID that's very unlikely to exist
		if pm.isProcessAlive(99999) {
			t.Error("non-existent process should not be alive")
		}
	})

	t.Run("init process", func(t *testing.T) {
		// PID 1 should always exist on Unix systems, but we might not have permission to signal it
		alive := pm.isProcessAlive(1)
		// Don't fail the test if we can't signal PID 1 due to permissions
		// This is expected behavior in containers or restricted environments
		_ = alive
	})
}

func TestSocketManagerBasics(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a custom socketManager for testing
	testSocketManager := &socketManager{
		path: filepath.Join(tempDir, SockName),
	}

	t.Run("listen and dial", func(t *testing.T) {
		// Start listening
		listener, err := testSocketManager.listen()
		if err != nil {
			t.Fatalf("listen failed: %v", err)
		}
		defer listener.Close()

		// Accept connections in background
		connCh := make(chan error, 1)
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				connCh <- err
				return
			}
			defer conn.Close()

			// Echo back what we receive
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				connCh <- err
				return
			}

			_, err = conn.Write(buf[:n])
			connCh <- err
		}()

		// Give listener time to start
		time.Sleep(10 * time.Millisecond)

		// Dial and send message
		conn, err := testSocketManager.dial()
		if err != nil {
			t.Fatalf("dial failed: %v", err)
		}
		defer conn.Close()

		testMsg := "hello"
		_, err = conn.Write([]byte(testMsg))
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}

		// Read echo
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}

		if string(buf[:n]) != testMsg {
			t.Errorf("got %q, expected %q", string(buf[:n]), testMsg)
		}

		// Check background goroutine
		if err := <-connCh; err != nil {
			t.Errorf("background connection error: %v", err)
		}
	})

	t.Run("dial without listener", func(t *testing.T) {
		_, err := testSocketManager.dial()
		if err == nil {
			t.Error("dial should fail when no listener exists")
		}
	})
}

func TestSendCommandIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a custom socketManager for testing
	testSocketManager := &socketManager{
		path: filepath.Join(tempDir, SockName),
	}

	t.Run("successful command with mock server", func(t *testing.T) {
		// Start a mock server
		listener, err := testSocketManager.listen()
		if err != nil {
			t.Fatalf("listen failed: %v", err)
		}
		defer listener.Close()

		// Handle connections in background
		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				go func(c net.Conn) {
					defer c.Close()

					buf := make([]byte, 2)
					n, err := c.Read(buf)
					if err != nil || n != 2 {
						return
					}

					cmd := buf[0]
					switch cmd {
					case 't':
						fmt.Fprint(c, "OK toggled\n")
					case 's':
						fmt.Fprint(c, "STATUS status=idle\n")
					case 'v':
						fmt.Fprintf(c, "STATUS proto=%s\n", ProtoVer)
					case 'q':
						fmt.Fprint(c, "OK quitting\n")
					default:
						fmt.Fprintf(c, "ERR unknown=%q\n", cmd)
					}
				}(conn)
			}
		}()

		// Give server time to start
		time.Sleep(10 * time.Millisecond)

		// Test different commands by manually creating connections
		tests := []struct {
			cmd      byte
			expected string
		}{
			{'t', "OK toggled\n"},
			{'s', "STATUS status=idle\n"},
			{'v', fmt.Sprintf("STATUS proto=%s\n", ProtoVer)},
			{'q', "OK quitting\n"},
			{'x', "ERR unknown='x'\n"},
		}

		for _, tt := range tests {
			conn, err := testSocketManager.dial()
			if err != nil {
				t.Errorf("dial failed for command %c: %v", tt.cmd, err)
				continue
			}

			_, err = conn.Write([]byte{tt.cmd, '\n'})
			if err != nil {
				conn.Close()
				t.Errorf("write failed for command %c: %v", tt.cmd, err)
				continue
			}

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			conn.Close()

			if err != nil {
				t.Errorf("read failed for command %c: %v", tt.cmd, err)
				continue
			}

			resp := string(buf[:n])
			if resp != tt.expected {
				t.Errorf("command %c: got %q, expected %q", tt.cmd, resp, tt.expected)
			}
		}
	})
}

func TestPathFunctions(t *testing.T) {
	t.Run("SockPath", func(t *testing.T) {
		path, err := SockPath()
		if err != nil {
			t.Fatalf("SockPath failed: %v", err)
		}

		if !filepath.IsAbs(path) {
			t.Error("SockPath should return absolute path")
		}

		if filepath.Base(path) != SockName {
			t.Errorf("SockPath should end with %s, got %s", SockName, filepath.Base(path))
		}
	})

	t.Run("getSockPath", func(t *testing.T) {
		path, err := getSockPath()
		if err != nil {
			t.Fatalf("getSockPath failed: %v", err)
		}

		if !filepath.IsAbs(path) {
			t.Error("getSockPath should return absolute path")
		}

		if filepath.Base(path) != SockName {
			t.Errorf("getSockPath should end with %s, got %s", SockName, filepath.Base(path))
		}
	})

	t.Run("getPidPath", func(t *testing.T) {
		path, err := getPidPath()
		if err != nil {
			t.Fatalf("getPidPath failed: %v", err)
		}

		if !filepath.IsAbs(path) {
			t.Error("getPidPath should return absolute path")
		}

		if filepath.Base(path) != PidName {
			t.Errorf("getPidPath should end with %s, got %s", PidName, filepath.Base(path))
		}
	})
}

func TestConstants(t *testing.T) {
	if SockName == "" {
		t.Error("SockName should not be empty")
	}
	if PidName == "" {
		t.Error("PidName should not be empty")
	}
	if ProtoVer == "" {
		t.Error("ProtoVer should not be empty")
	}
}

// Test the public API functions with temporary directories
func TestPublicAPIWithTempDirs(t *testing.T) {
	// We can't override the internal functions, but we can test the public API
	// and clean up any files we create

	t.Run("CheckExistingDaemon with no daemon", func(t *testing.T) {
		// This should succeed when no daemon is running
		// Clean up any existing PID file first
		pidPath, _ := getPidPath()
		os.Remove(pidPath)

		err := CheckExistingDaemon()
		if err != nil {
			t.Errorf("CheckExistingDaemon should succeed when no daemon running: %v", err)
		}
	})

	t.Run("CreatePidFile and RemovePidFile", func(t *testing.T) {
		// Clean up first
		pidPath, _ := getPidPath()
		os.Remove(pidPath)

		err := CreatePidFile()
		if err != nil {
			t.Fatalf("CreatePidFile failed: %v", err)
		}

		// Check file exists
		if _, err := os.Stat(pidPath); os.IsNotExist(err) {
			t.Error("PID file should exist after CreatePidFile")
		}

		err = RemovePidFile()
		if err != nil {
			t.Fatalf("RemovePidFile failed: %v", err)
		}

		// Check file is removed
		if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
			t.Error("PID file should not exist after RemovePidFile")
		}
	})
}
