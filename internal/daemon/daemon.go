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

type Daemon struct {
	mu       sync.RWMutex
	notifier notify.Notifier

	ctx    context.Context
	cancel context.CancelFunc

	pipeline pipeline.Pipeline
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
	}

	return d
}

func (d *Daemon) status() pipeline.Status {
	if d.pipeline == nil {
		return pipeline.Idle
	}
	return d.pipeline.Status()
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
		status := d.status()
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
	var actionChan chan<- pipeline.Action

	switch d.status() {
	case pipeline.Idle:
		p := pipeline.New()
		p.Run(d.ctx)
		d.pipeline = p
		go d.notifier.RecordingStarted()

	case pipeline.Recording:
		go d.notifier.Aborted()
		if d.pipeline != nil {
			d.pipeline.Stop() // aborted during recording (chunks not sent to transcriber yet)
			d.pipeline = nil
		}

	case pipeline.Transcribing:
		go d.notifier.RecordingEnded()
		actionChan = d.pipeline.Actions()
		actionChan <- pipeline.Inject

	case pipeline.Injecting:
		go d.notifier.Aborted()

		if d.pipeline != nil {
			d.pipeline.Stop() // aborted during injection
			d.pipeline = nil
		}

	default:
		d.mu.Unlock()
	}
}
