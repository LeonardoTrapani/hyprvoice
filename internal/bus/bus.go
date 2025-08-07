package bus

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

const (
	SockName = "control.sock"
	PidName  = "hyprvoice.pid"
	ProtoVer = "0.1"
)

type pidManager struct {
	path string
}

func newPidManager() (*pidManager, error) {
	pidPath, err := getPidPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get PID path: %w", err)
	}
	return &pidManager{path: pidPath}, nil
}

func (pm *pidManager) checkExisting() error {
	log.Printf("Checking for existing daemon at: %s", pm.path)

	pidData, err := os.ReadFile(pm.path)
	if os.IsNotExist(err) {
		log.Printf("No PID file found, daemon not running")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error reading PID file: %w", err)
	}

	log.Printf("Found PID file with content: %s", string(pidData))

	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		log.Printf("Invalid PID in file, removing stale PID file: %v", err)
		pm.removeStaleFile()
		return nil
	}

	if pm.isProcessAlive(pid) {
		log.Printf("Process %d is alive, daemon already running", pid)
		return fmt.Errorf("daemon already running with PID %d", pid)
	}

	log.Printf("Process %d not alive, removing stale PID file", pid)
	pm.removeStaleFile()
	return nil
}

func (pm *pidManager) create() error {
	if err := os.MkdirAll(filepath.Dir(pm.path), 0o700); err != nil {
		return fmt.Errorf("failed to create PID directory: %w", err)
	}

	pid := os.Getpid()
	log.Printf("Creating PID file at %s with PID %d", pm.path, pid)

	err := os.WriteFile(pm.path, []byte(strconv.Itoa(pid)), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

func (pm *pidManager) remove() error {
	log.Printf("Removing PID file: %s", pm.path)
	if err := os.Remove(pm.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}

func (pm *pidManager) isProcessAlive(pid int) bool {
	log.Printf("Checking if process %d is alive", pid)

	proc, err := os.FindProcess(pid)
	if err != nil {
		log.Printf("Process %d not found: %v", pid, err)
		return false
	}

	err = proc.Signal(syscall.Signal(0))
	if err != nil {
		log.Printf("Process %d not alive (signal failed: %v)", pid, err)
		return false
	}

	return true
}

func (pm *pidManager) removeStaleFile() {
	if err := os.Remove(pm.path); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to remove stale PID file: %v", err)
	}
}

type socketManager struct {
	path string
}

func newSocketManager() (*socketManager, error) {
	sockPath, err := getSockPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get socket path: %w", err)
	}
	return &socketManager{path: sockPath}, nil
}

func (sm *socketManager) listen() (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(sm.path), 0o700); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	os.Remove(sm.path)

	listener, err := net.Listen("unix", sm.path)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on socket %s: %w", sm.path, err)
	}

	return listener, nil
}

func (sm *socketManager) dial() (net.Conn, error) {
	conn, err := net.Dial("unix", sm.path)
	if err != nil {
		return nil, fmt.Errorf("failed to dial socket %s: %w", sm.path, err)
	}
	return conn, nil
}

func getSockPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hyprvoice", SockName), nil
}

func getPidPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hyprvoice", PidName), nil
}

func SockPath() (string, error) {
	return getSockPath()
}

func Listen() (net.Listener, error) {
	sm, err := newSocketManager()
	if err != nil {
		return nil, err
	}
	return sm.listen()
}

func Dial() (net.Conn, error) {
	sm, err := newSocketManager()
	if err != nil {
		return nil, err
	}
	return sm.dial()
}

func CheckExistingDaemon() error {
	pm, err := newPidManager()
	if err != nil {
		return err
	}
	return pm.checkExisting()
}

func CreatePidFile() error {
	pm, err := newPidManager()
	if err != nil {
		return err
	}
	return pm.create()
}

func RemovePidFile() error {
	pm, err := newPidManager()
	if err != nil {
		return err
	}
	return pm.remove()
}

func SendCommand(cmd byte) (string, error) {
	c, err := Dial()
	if err != nil {
		return "", fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer c.Close()

	_, err = c.Write([]byte{cmd, '\n'})
	if err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	resp, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return resp, nil
}
