package hotkeydaemon

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
	"time"

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
	// Check if daemon is already running
	if err := bus.CheckExistingDaemon(); err != nil {
		return err
	}

	ln, err := bus.Listen()
	if err != nil {
		return err
	}
	defer ln.Close()

	// Create PID file
	if err := bus.CreatePidFile(); err != nil {
		return fmt.Errorf("failed to create PID file: %w", err)
	}
	defer bus.RemovePidFile()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, shutting down gracefully", sig)
		d.cancel()
	}()

	log.Printf("Daemon started, listening on socket")
	for {
		select {
		case <-d.ctx.Done():
			log.Printf("Shutdown requested, exiting")
			return nil
		default:
		}

		// Set a timeout for Accept to make it cancellable
		if tcpListener, ok := ln.(*net.UnixListener); ok {
			tcpListener.SetDeadline(time.Now().Add(100 * time.Millisecond))
		}

		c, err := ln.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // timeout, check for shutdown
			}
			log.Printf("Accept error: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		go d.handle(c)
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

	d.mu.Lock()
	defer d.mu.Unlock()

	switch cmd {
	case 't': // toggle
		d.recording = !d.recording
		d.notifier.RecordingChanged(d.recording)
		log.Printf("Recording toggled: %t", d.recording)
		fmt.Fprintf(c, "STATUS recording=%t\n", d.recording)
	case 's': // status
		fmt.Fprintf(c, "STATUS recording=%t\n", d.recording)
	case 'v': // protocol version
		fmt.Fprintf(c, "STATUS proto=%s\n", bus.ProtoVer)
	case 'q': // quit daemon
		log.Printf("Shutdown requested")
		fmt.Fprint(c, "OK quitting\n")
		go func() {
			time.Sleep(100 * time.Millisecond) // give time for client to read
			d.cancel()                         // trigger graceful shutdown
		}()
	default:
		log.Printf("Unknown command: %c", cmd)
		fmt.Fprintf(c, "ERR unknown=%q\n", cmd)
	}
}
