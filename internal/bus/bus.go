package bus

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

const SockName = "control.sock"
const PidName = "hyprvoice.pid"
const ProtoVer = "0.1"

// ~/.cache/hyprvoice/control.sock
func SockPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	hd := filepath.Join(dir, "hyprvoice")
	return filepath.Join(hd, SockName), nil
}

// ~/.cache/hyprvoice/hyprvoice.pid
func PidPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	hd := filepath.Join(dir, "hyprvoice")
	return filepath.Join(hd, PidName), nil
}

func Listen() (net.Listener, error) {
	sp, err := SockPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(sp), 0o700); err != nil {
		return nil, err
	}
	_ = os.Remove(sp) // stale socket from last run
	return net.Listen("unix", sp)
}

func Dial() (net.Conn, error) {
	sp, err := SockPath()
	if err != nil {
		return nil, err
	}
	return net.Dial("unix", sp)
}

func SendCommand(cmd byte) (string, error) {
	c, err := Dial()
	if err != nil {
		return "", err
	}
	defer c.Close()

	_, err = c.Write([]byte{cmd, '\n'})
	if err != nil {
		return "", err
	}

	resp, err := bufio.NewReader(c).ReadString('\n')
	return resp, err
}

func CheckExistingDaemon() error {
	pidPath, err := PidPath()
	if err != nil {
		return err
	}

	pidData, err := os.ReadFile(pidPath)
	if os.IsNotExist(err) {
		return nil // no existing daemon
	}
	if err != nil {
		return err
	}

	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return nil // invalid pid file, assume stale
	}

	// Check if process exists
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}

	// Try to signal the process to check if it's alive
	if err := proc.Signal(os.Signal(nil)); err != nil {
		return nil // process not alive, stale pid file
	}

	return fmt.Errorf("daemon already running with PID %d", pid)
}

func CreatePidFile() error {
	pidPath, err := PidPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(pidPath), 0o700); err != nil {
		return err
	}

	pid := os.Getpid()
	return os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0o600)
}

func RemovePidFile() error {
	pidPath, err := PidPath()
	if err != nil {
		return err
	}
	return os.Remove(pidPath)
}
