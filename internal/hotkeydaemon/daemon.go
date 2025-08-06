package hotkeydaemon

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/bus"
	"github.com/leonardotrapani/hyprvoice/internal/notify"
)

type Daemon struct {
	mu        sync.Mutex
	recording bool
	notifier  notify.Notifier
}

func New(n notify.Notifier) *Daemon {
	if n == nil {
		n = notify.Desktop{}
	}
	return &Daemon{
		notifier: n,
	}
}

func (d *Daemon) Rec() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.recording
}

func (d *Daemon) Run() error {
	ln, err := bus.Listen()
	if err != nil {
		return err
	}
	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		go d.handle(c)
	}
}

func (d *Daemon) handle(c net.Conn) {
	defer c.Close()

	line, _ := bufio.NewReader(c).ReadString('\n')
	if len(line) == 0 {
		fmt.Fprint(c, "ERR empty\n")
		return
	}
	cmd := line[0]

	d.mu.Lock()
	defer d.mu.Unlock()

	switch cmd {
	case 't': // toggle
		d.recording = !d.recording
		d.notifier.RecordingChanged(d.recording)
		fmt.Fprintf(c, "STATUS recording=%t\n", d.recording)
	case 's': // status
		fmt.Fprintf(c, "STATUS recording=%t\n", d.recording)
	case 'v': // protocol version
		fmt.Fprintf(c, "STATUS proto=%s\n", bus.ProtoVer)
	case 'q': // quit daemon
		fmt.Fprint(c, "OK quitting\n")
		go func() {
			time.Sleep(100 * time.Millisecond) // give time for client to read
			panic("daemon exit requested")
		}()
	default:
		fmt.Fprintf(c, "ERR unknown=%q\n", cmd)
	}
}
