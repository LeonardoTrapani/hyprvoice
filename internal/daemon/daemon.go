package daemon

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/leonardotrapani/hyprvoice/internal/bus"
	"github.com/leonardotrapani/hyprvoice/internal/notify"
)

type Daemon struct {
	mu        sync.Mutex
	recording bool
	notifier  notify.Notifier
	ctx       context.Context
	cancel    context.CancelFunc
}

func New(n notify.Notifier) *Daemon {
	if n == nil {
		n = notify.Desktop{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Daemon{
		notifier: n,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (d *Daemon) Rec() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.recording
}

func (d *Daemon) Run() error {
	if err := bus.CheckExistingDaemon(); err != nil {
		return err
	}

	ln, err := bus.Listen()
	if err != nil {
		return err
	}
	defer ln.Close()

	if err := bus.CreatePidFile(); err != nil {
		return fmt.Errorf("failed to create PID file: %w", err)
	}
	defer bus.RemovePidFile()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigCh)

	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, shutting down gracefully", sig)
		d.cancel()
	}()

	log.Printf("Daemon started, listening on socket")

	// Accept connections in a goroutine
	connCh := make(chan net.Conn)
	errCh := make(chan error)

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				errCh <- err
				return
			}
			connCh <- c
		}
	}()

	for {
		select {
		case <-d.ctx.Done():
			log.Printf("Shutdown requested, exiting")
			return nil
		case c := <-connCh:
			go d.handle(c)
		case err := <-errCh:
			// If context is cancelled, this is expected
			if d.ctx.Err() != nil {
				return nil
			}
			log.Printf("Accept error: %v", err)
			return fmt.Errorf("accept failed: %w", err)
		}
	}
}

func (d *Daemon) handle(c net.Conn) {
	defer c.Close()

	line, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		log.Printf("Client read error: %v", err)
		fmt.Fprintf(c, "ERR read_error: %v\n", err)
		return
	}
	if len(line) == 0 {
		fmt.Fprint(c, "ERR empty\n")
		return
	}
	cmd := line[0]

	switch cmd {
	case 't': // toggle
		d.mu.Lock()
		d.recording = !d.recording
		isRecording := d.recording
		d.mu.Unlock()

		d.notifier.RecordingChanged(isRecording)
		log.Printf("Recording toggled: %t", isRecording)

		fmt.Fprintf(c, "STATUS recording=%t\n", isRecording)
	case 's': // status
		d.mu.Lock()
		isRecording := d.recording
		d.mu.Unlock()

		fmt.Fprintf(c, "STATUS recording=%t\n", isRecording)
	case 'v': // protocol version
		fmt.Fprintf(c, "STATUS proto=%s\n", bus.ProtoVer)
	case 'q': // quit daemon
		log.Printf("Shutdown requested")
		fmt.Fprint(c, "OK quitting\n")
		d.cancel()
	default:
		log.Printf("Unknown command: %c", cmd)
		fmt.Fprintf(c, "ERR unknown=%q\n", cmd)
	}
}
