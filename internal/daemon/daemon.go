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
	"time"

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

	pipeline       pipeline.Pipeline
	pipelineCancel context.CancelFunc
	statusCh       <-chan Status
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
	go func() {
		defer func() {
			// Always clean up when this goroutine exits
			d.mu.Lock()
			d.status = Idle
			d.statusCh = nil
			d.pipeline = nil
			d.pipelineCancel = nil
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
	}()
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
		fmt.Fprintf(c, "STATUS recording=%s\n", status)
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
	d.mu.Lock()
	defer d.mu.Unlock()

	var notification func()

	switch d.status {
	case Idle:
		ctx, cancel := context.WithTimeout(d.ctx, 5*time.Minute)
		p := pipeline.New()
		d.pipeline = p
		d.pipelineCancel = cancel
		d.statusCh = p.Run(ctx)
		notification = d.notifier.RecordingStarted

		// Start status reader for this pipeline
		d.startStatusReader(ctx, d.statusCh)

	case Recording:
		if d.pipelineCancel != nil {
			d.pipelineCancel() // Context cleanup handles the rest
		}
		notification = d.notifier.RecordingEnded

	case Transcribing:
		if d.pipeline != nil {
			d.pipeline.Inject()
		}
		// No notification for injection start

	case Injecting:
		if d.pipelineCancel != nil {
			d.pipelineCancel() // Context cleanup handles the rest
		}
		notification = d.notifier.RecordingEnded
	}

	// Send notification after releasing lock
	if notification != nil {
		go notification()
	}
}
