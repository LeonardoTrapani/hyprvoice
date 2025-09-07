package recording

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type AudioFrame struct {
	Data      []byte
	Timestamp time.Time
}

type Config struct {
	SampleRate        int
	Channels          int
	Format            string
	BufferSize        int
	Device            string
	ChannelBufferSize int
	Timeout           time.Duration
}

type Recorder struct {
	config    Config
	recording atomic.Bool

	mu     sync.Mutex // guards cmd and cancel
	cmd    *exec.Cmd
	cancel context.CancelFunc

	wg sync.WaitGroup
}

func NewRecorder(config Config) *Recorder {
	return &Recorder{config: config}
}

func (r *Recorder) IsRecording() bool {
	return r.recording.Load()
}

func (r *Recorder) Start(ctx context.Context) (<-chan AudioFrame, <-chan error, error) {
	if r.recording.Load() {
		return nil, nil, fmt.Errorf("already recording")
	}

	if err := r.validateConfig(); err != nil {
		return nil, nil, err
	}

	if err := CheckPipeWireAvailable(ctx); err != nil {
		return nil, nil, fmt.Errorf("PipeWire not available: %w", err)
	}

	recordingCtx, cancel := context.WithCancel(ctx)

	frameCh := make(chan AudioFrame, r.config.ChannelBufferSize)
	errCh := make(chan error, 1)

	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()

	r.recording.Store(true)
	r.wg.Add(1)
	go r.captureLoop(recordingCtx, frameCh, errCh)

	return frameCh, errCh, nil
}

func (r *Recorder) Stop() {
	if !r.recording.Load() {
		return
	}

	r.requestCancel()

	r.wg.Wait()
}

func (r *Recorder) captureLoop(ctx context.Context, frameCh chan<- AudioFrame, errCh chan<- error) {
	defer func() {
		close(frameCh)
		close(errCh)
		r.recording.Store(false)

		r.mu.Lock()
		r.cancel = nil
		r.mu.Unlock()

		r.wg.Done()
	}()

	args := r.buildPwRecordArgs()
	cmd := exec.CommandContext(ctx, "pw-record", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.emitErr(errCh, fmt.Errorf("create stdout pipe: %w", err))
		r.requestCancel()
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		r.emitErr(errCh, fmt.Errorf("create stderr pipe: %w", err))
		r.requestCancel()
		return
	}

	if err := cmd.Start(); err != nil {
		r.emitErr(errCh, fmt.Errorf("start pw-record: %w", err))
		r.requestCancel()
		return
	}

	r.mu.Lock()
	r.cmd = cmd
	r.mu.Unlock()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("Recording stderr: %s", scanner.Text())
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			buffer := make([]byte, r.config.BufferSize)
			var sentCount int
			var droppedCount int
			lastDropLog := time.Now()
			n, readErr := stdout.Read(buffer)
			if n > 0 {
				frameData := make([]byte, n)
				copy(frameData, buffer[:n])

				frame := AudioFrame{Data: frameData, Timestamp: time.Now()}

				select {
				case frameCh <- frame:
					sentCount++
				case <-ctx.Done():
					return
				default:
					droppedCount++
					if time.Since(lastDropLog) > time.Second {
						log.Printf("Recording: dropped %d frames due to backpressure", droppedCount)
						lastDropLog = time.Now()
						droppedCount = 0
					}
				}
			}

			if readErr != nil {
				if errors.Is(readErr, io.EOF) {
					return
				}
				r.emitErr(errCh, fmt.Errorf("read audio: %w", readErr))
				r.requestCancel()
				return
			}

		}
	}
}

func (r *Recorder) requestCancel() {
	r.mu.Lock()
	cancel := r.cancel
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (r *Recorder) emitErr(errCh chan<- error, err error) {
	select {
	case errCh <- err:
	default:
	}
	log.Printf("Recording error: %v", err)
}

func (r *Recorder) buildPwRecordArgs() []string {
	args := []string{
		"--format", r.config.Format,
		"--rate", strconv.Itoa(r.config.SampleRate),
		"--channels", strconv.Itoa(r.config.Channels),
		"-", // stdout
	}
	if r.config.Device != "" {
		args = append(args, "--target", r.config.Device)
	}
	return args
}

func CheckPipeWireAvailable(ctx context.Context) error {
	if _, err := exec.LookPath("pw-record"); err != nil {
		return fmt.Errorf("pw-record not found: %w (install pipewire-tools)", err)
	}
	// Use a short timeout to avoid hangs on misconfigured systems.
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(checkCtx, "pw-cli", "info", "all")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("PipeWire not running or accessible: %w", err)
	}
	return nil
}

func (r *Recorder) validateConfig() error {
	if r.config.SampleRate <= 0 {
		return fmt.Errorf("invalid SampleRate: %d", r.config.SampleRate)
	}
	if r.config.Channels <= 0 {
		return fmt.Errorf("invalid Channels: %d", r.config.Channels)
	}
	if r.config.BufferSize <= 0 {
		return fmt.Errorf("invalid BufferSize: %d", r.config.BufferSize)
	}
	if r.config.ChannelBufferSize <= 0 {
		return fmt.Errorf("invalid ChannelBufferSize: %d", r.config.ChannelBufferSize)
	}
	if r.config.Format == "" {
		return fmt.Errorf("invalid Format: empty")
	}
	// For s16, sample frame size is 2 bytes per sample per channel.
	if r.config.Format == "s16" {
		frameBytes := 2 * r.config.Channels
		if r.config.BufferSize%frameBytes != 0 {
			log.Printf("Recording: BufferSize %d not aligned to frame size %d; audio frames may split",
				r.config.BufferSize, frameBytes)
		}
	}
	return nil
}
