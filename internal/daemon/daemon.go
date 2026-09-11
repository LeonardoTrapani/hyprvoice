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
	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/notify"
	"github.com/leonardotrapani/hyprvoice/internal/pipeline"
)

type Daemon struct {
	mu        sync.RWMutex
	notifier  notify.Notifier
	notifType string
	configMgr *config.Manager

	ctx    context.Context
	cancel context.CancelFunc

	pipeline pipeline.Pipeline

	wg       sync.WaitGroup
	monitors sync.WaitGroup
}

func New() (*Daemon, error) {
	configMgr, err := config.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	conf := configMgr.GetConfig()
	ctx, cancel := context.WithCancel(context.Background())

	// force desktop notifications when legacy config so user sees the onboarding prompt
	notifType := notificationType(conf, configMgr.IsLegacy())

	d := &Daemon{
		notifier:  notify.NewNotifier(notifType, conf.Notifications.Messages.Resolve()),
		notifType: notifType,
		configMgr: configMgr,
		ctx:       ctx,
		cancel:    cancel,
	}

	return d, nil
}

func (d *Daemon) onConfigReload() {
	log.Printf("Config reloaded, restarting pipeline")
	d.stopPipeline()

	d.replaceNotifier(d.configMgr.GetConfig())
	d.sendNotify(notify.MsgConfigReloaded)
}

func notificationType(conf *config.Config, legacy bool) string {
	if legacy {
		return "desktop"
	}
	if !conf.Notifications.Enabled {
		return "none"
	}
	return conf.Notifications.Type
}

func (d *Daemon) replaceNotifier(conf *config.Config) {
	nextType := notificationType(conf, d.configMgr.IsLegacy())
	msgs := conf.Notifications.Messages.Resolve()

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.notifType == nextType {
		d.notifier.SetMessages(msgs)
		return
	}
	d.notifier.BeginSession()
	d.notifier = notify.NewNotifier(nextType, msgs)
	d.notifType = nextType
}

func (d *Daemon) status() pipeline.Status {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.pipeline == nil {
		return pipeline.Idle
	}
	return d.pipeline.Status()
}

func (d *Daemon) stopPipeline() pipeline.Pipeline {
	d.mu.Lock()
	p := d.pipeline
	d.pipeline = nil
	d.mu.Unlock()

	if p != nil {
		p.Stop()
	}
	d.monitors.Wait()
	return p
}

func (d *Daemon) Run() error {
	if err := bus.CheckExistingDaemon(); err != nil {
		return err
	}

	d.configMgr.SetOnConfigReload(d.onConfigReload)

	ln, err := bus.Listen()
	if err != nil {
		return err
	}
	defer ln.Close()

	if err := bus.CreatePidFile(); err != nil {
		return fmt.Errorf("failed to create PID file: %w", err)
	}
	defer bus.RemovePidFile()

	if err := d.configMgr.StartWatching(d.ctx); err != nil {
		log.Printf("Warning: failed to start config file watching: %v", err)
	}
	defer d.configMgr.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigCh)

	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, shutting down gracefully", sig)
		d.cancel()
	}()

	go func() {
		<-d.ctx.Done()
		if err := ln.Close(); err != nil {
			log.Printf("Error closing listener: %v", err)
		}
	}()

	log.Printf("Daemon started, listening on socket")

	for {
		c, err := ln.Accept()
		if err != nil {
			if d.ctx.Err() != nil {
				log.Printf("Shutdown requested, waiting for connections to finish")
				d.wg.Wait()
				return nil
			}
			log.Printf("Accept error: %v", err)
			return fmt.Errorf("accept failed: %w", err)
		}
		d.wg.Add(1)
		go d.handle(c)
	}
}

func (d *Daemon) handle(c net.Conn) {
	defer c.Close()
	defer d.wg.Done()

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
	case 'c':
		d.cancelPipeline()
		fmt.Fprint(c, "OK cancelled\n")
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
	if d.configMgr.IsLegacy() {
		d.sendNotifyError("Legacy config detected. Run: hyprvoice onboarding")
		return
	}
	conf := d.configMgr.GetConfig()
	switch d.status() {
	case pipeline.Idle:
		d.stopPipeline()
		d.beginNotifySession()

		p := pipeline.New(conf)
		p.Run(d.ctx)

		d.mu.Lock()
		d.pipeline = p
		d.mu.Unlock()

		d.sendNotify(notify.MsgRecordingStarted)
		d.startMonitors(p)

	case pipeline.Recording:
		d.stopPipeline()
		d.sendNotify(notify.MsgRecordingAborted)

	case pipeline.Transcribing:
		d.mu.RLock()
		if d.pipeline != nil {
			actionChan := d.pipeline.GetActionCh()
			log.Printf("Daemon: Sending inject action to pipeline")
			d.mu.RUnlock()
			actionChan <- pipeline.Inject
		} else {
			d.mu.RUnlock()
		}
		d.sendNotify(notify.MsgTranscribing)

	case pipeline.Injecting:
		p := d.stopPipeline()
		if p == nil || !p.Succeeded() {
			d.sendNotify(notify.MsgInjectionAborted)
		}
	}
}

func (d *Daemon) cancelPipeline() {
	switch d.status() {
	case pipeline.Idle:
		log.Printf("Daemon: Cancel requested but pipeline is idle, ignoring")
	default:
		d.stopPipeline()
		d.sendNotify(notify.MsgOperationCancelled)
	}
}

func (d *Daemon) sendNotify(mt notify.MessageType) {
	d.mu.RLock()
	n := d.notifier
	d.mu.RUnlock()
	n.Send(mt)
}

func (d *Daemon) sendNotifyError(msg string) {
	d.mu.RLock()
	n := d.notifier
	d.mu.RUnlock()
	n.Error(msg)
}

func (d *Daemon) beginNotifySession() {
	d.mu.RLock()
	n := d.notifier
	d.mu.RUnlock()
	n.BeginSession()
}

func (d *Daemon) startMonitors(p pipeline.Pipeline) {
	d.monitors.Add(2)
	go func() {
		defer d.monitors.Done()
		d.monitorPipelineErrors(p)
	}()
	go func() {
		defer d.monitors.Done()
		d.monitorPipelineNotifications(p)
	}()
}

func (d *Daemon) monitorPipelineErrors(p pipeline.Pipeline) {
	errorCh := p.GetErrorCh()
	for {
		select {
		case pipelineErr, ok := <-errorCh:
			if !ok {
				return
			}
			message := pipelineErr.Message

			if pipelineErr.Err != nil {
				message = fmt.Sprintf("%s: %v", message, pipelineErr.Err)
			}

			d.sendNotifyError(message)
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *Daemon) monitorPipelineNotifications(p pipeline.Pipeline) {
	notifyCh := p.GetNotifyCh()
	for {
		select {
		case mt, ok := <-notifyCh:
			if !ok {
				return
			}
			d.sendNotify(mt)
		case <-d.ctx.Done():
			return
		}
	}
}
