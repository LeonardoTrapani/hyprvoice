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
	"github.com/leonardotrapani/hyprvoice/internal/pipeline"
)

type Status = pipeline.Status

const (
	Idle         = pipeline.Idle
	Recording    = pipeline.Recording
	Transcribing = pipeline.Transcribing
	Injecting    = pipeline.Injecting
)

type Daemon struct {
	mu       sync.RWMutex
	status   Status
	notifier notify.Notifier

	ctx    context.Context
	cancel context.CancelFunc

	pipeline      pipeline.Pipeline
	actionChannel chan<- pipeline.Action
}

func New(n notify.Notifier) *Daemon {
	if n == nil {
		n = notify.Desktop{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	d := &Daemon{
		notifier: n,
		ctx:      ctx,
		cancel:   cancel,
		status:   Idle,
	}

	return d
}

func (d *Daemon) Status() Status {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.status
}

func (d *Daemon) startStatusReader(ctx context.Context, statusCh <-chan Status) {
	defer func() {
		// Always clean up when this goroutine exits
		d.mu.Lock()
		d.status = Idle
		d.pipeline = nil
		d.actionChannel = nil
		d.mu.Unlock()
	}()

	for {
		select {
		case status, ok := <-statusCh:
			if !ok {
				return // Channel closed
			}

			d.mu.Lock()
			oldStatus := d.status
			d.status = status
			d.mu.Unlock()

			if oldStatus != status {
				log.Printf("Status changed: %s -> %s", oldStatus, status)
			}

		case <-ctx.Done():
			return // Context cancelled
		}
	}
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

	// Close the listener when context is done
	go func() {
		<-d.ctx.Done()
		ln.Close()
	}()

	log.Printf("Daemon started, listening on socket")

	for {
		c, err := ln.Accept()
		if err != nil {
			if d.ctx.Err() != nil {
				log.Printf("Shutdown requested")
				return nil
			}
			log.Printf("Accept error: %v", err)
			return fmt.Errorf("accept failed: %w", err)
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

	switch cmd {
	case 't':
		d.toggle()
		fmt.Fprint(c, "OK toggled\n")
	case 's':
		status := d.Status()
		fmt.Fprintf(c, "STATUS status=%s\n", status)
	case 'v':
		fmt.Fprintf(c, "STATUS proto=%s\n", bus.ProtoVer)
	case 'q':
		fmt.Fprint(c, "OK quitting\n")
		d.cancel()
	default:
		log.Printf("Unknown command: %c", cmd)
		fmt.Fprintf(c, "ERR unknown=%q\n", cmd)
	}
}

func (d *Daemon) toggle() {
	switch d.status {
	case Idle:
		// Defensive: if a prior pipeline exists, stop and wait before starting a new one
		if d.pipeline != nil {
			d.pipeline.Stop()
			d.pipeline = nil
		}

		p := pipeline.New()
		statusCh, actionCh := p.Run(d.ctx)

		d.pipeline = p
		d.actionChannel = actionCh

		go d.notifier.RecordingStarted()

		go d.startStatusReader(d.ctx, statusCh)

	case Recording:
		go d.notifier.RecordingEnded()
		d.pipeline.Stop()

	case Transcribing:
		select {
		case d.actionChannel <- pipeline.Inject:
		default:
		}

	case Injecting:
		go d.notifier.RecordingEnded()
		d.pipeline.Stop() // aborted during injection
	}
}
