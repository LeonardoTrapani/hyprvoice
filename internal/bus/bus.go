package bus

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
)

const SockName = "control.sock"
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
