package pipeline

import (
	"context"
	"testing"
	"time"
)

func TestPipelineStatus(t *testing.T) {
	p := New()

	t.Run("initial status", func(t *testing.T) {
		status := p.Status()
		if status != "" && status != Idle {
			t.Errorf("initial status should be empty or Idle, got %s", status)
		}
	})

	t.Run("status changes", func(t *testing.T) {
		pipeline := p.(*pipeline)

		pipeline.setStatus(Recording)
		if p.Status() != Recording {
			t.Errorf("status should be Recording, got %s", p.Status())
		}

		pipeline.setStatus(Transcribing)
		if p.Status() != Transcribing {
			t.Errorf("status should be Transcribing, got %s", p.Status())
		}

		pipeline.setStatus(Injecting)
		if p.Status() != Injecting {
			t.Errorf("status should be Injecting, got %s", p.Status())
		}

		pipeline.setStatus(Idle)
		if p.Status() != Idle {
			t.Errorf("status should be Idle, got %s", p.Status())
		}
	})
}

func TestPipelineChannels(t *testing.T) {
	p := New()

	t.Run("action channel", func(t *testing.T) {
		actionCh := p.GetActionCh()
		if actionCh == nil {
			t.Error("action channel should not be nil")
		}

		// Test non-blocking send
		select {
		case actionCh <- Inject:
		default:
			t.Error("action channel should accept at least one message")
		}
	})

	t.Run("error channel", func(t *testing.T) {
		errorCh := p.GetErrorCh()
		if errorCh == nil {
			t.Error("error channel should not be nil")
		}

		// Should be empty initially
		select {
		case <-errorCh:
			t.Error("error channel should be empty initially")
		default:
		}
	})
}

func TestPipelineErrorHandling(t *testing.T) {
	p := New().(*pipeline)

	t.Run("send error", func(t *testing.T) {
		testTitle := "Test Error"
		testMessage := "This is a test error"
		testErr := context.Canceled

		p.sendError(testTitle, testMessage, testErr)

		select {
		case pipelineErr := <-p.GetErrorCh():
			if pipelineErr.Title != testTitle {
				t.Errorf("error title should be %q, got %q", testTitle, pipelineErr.Title)
			}
			if pipelineErr.Message != testMessage {
				t.Errorf("error message should be %q, got %q", testMessage, pipelineErr.Message)
			}
			if pipelineErr.Err != testErr {
				t.Errorf("error should be %v, got %v", testErr, pipelineErr.Err)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("should receive error on error channel")
		}
	})

	t.Run("error channel full", func(t *testing.T) {
		// Fill up the error channel (capacity is 10)
		for i := 0; i < 10; i++ {
			p.sendError("Test", "Test message", nil)
		}

		// This should not block (it should drop the error)
		done := make(chan bool, 1)
		go func() {
			p.sendError("Overflow", "This should be dropped", nil)
			done <- true
		}()

		select {
		case <-done:
			// Success - the send didn't block
		case <-time.After(100 * time.Millisecond):
			t.Error("sendError should not block when channel is full")
		}
	})
}

func TestPipelineLifecycle(t *testing.T) {
	t.Run("multiple Run calls", func(t *testing.T) {
		p := New()
		ctx := context.Background()

		// First Run should work
		p.Run(ctx)

		// Second Run should be ignored (already running)
		p.Run(ctx)

		// Stop should work
		p.Stop()
	})

	t.Run("stop before run", func(t *testing.T) {
		p := New()

		// Stop before run should not panic
		p.Stop()
	})

	t.Run("multiple stops", func(t *testing.T) {
		p := New()
		ctx := context.Background()

		p.Run(ctx)
		p.Stop()

		// Second stop should not panic
		p.Stop()
	})
}

func TestPipelineContextCancellation(t *testing.T) {
	t.Run("context cancelled during run", func(t *testing.T) {
		p := New()
		ctx, cancel := context.WithCancel(context.Background())

		// Start pipeline
		p.Run(ctx)

		// Cancel context immediately
		cancel()

		// Wait for pipeline to stop
		p.Stop()

		// Pipeline should be idle
		if p.Status() != Idle {
			t.Errorf("pipeline should be idle after context cancellation, got %s", p.Status())
		}
	})

	t.Run("timeout context", func(t *testing.T) {
		p := New()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		// Start pipeline
		p.Run(ctx)

		// Wait for timeout
		time.Sleep(50 * time.Millisecond)

		// Pipeline should stop automatically
		p.Stop()

		if p.Status() != Idle {
			t.Errorf("pipeline should be idle after timeout, got %s", p.Status())
		}
	})
}

func TestPipelineConstants(t *testing.T) {
	t.Run("status constants", func(t *testing.T) {
		statuses := []Status{Idle, Recording, Transcribing, Injecting}
		for _, status := range statuses {
			if string(status) == "" {
				t.Errorf("status %v should have a string representation", status)
			}
		}
	})

	t.Run("action constants", func(t *testing.T) {
		actions := []Action{Inject}
		for _, action := range actions {
			if string(action) == "" {
				t.Errorf("action %v should have a string representation", action)
			}
		}
	})
}

func TestPipelineError(t *testing.T) {
	t.Run("pipeline error struct", func(t *testing.T) {
		title := "Test Title"
		message := "Test Message"
		err := context.Canceled

		pipelineErr := PipelineError{
			Title:   title,
			Message: message,
			Err:     err,
		}

		if pipelineErr.Title != title {
			t.Errorf("title should be %q, got %q", title, pipelineErr.Title)
		}
		if pipelineErr.Message != message {
			t.Errorf("message should be %q, got %q", message, pipelineErr.Message)
		}
		if pipelineErr.Err != err {
			t.Errorf("err should be %v, got %v", err, pipelineErr.Err)
		}
	})

	t.Run("pipeline error with nil error", func(t *testing.T) {
		pipelineErr := PipelineError{
			Title:   "Test",
			Message: "Test message",
			Err:     nil,
		}

		if pipelineErr.Err != nil {
			t.Error("error should be nil")
		}
	})
}

func TestPipelineConcurrency(t *testing.T) {
	t.Run("concurrent status reads", func(t *testing.T) {
		p := New()
		ctx := context.Background()

		p.Run(ctx)
		defer p.Stop()

		// Read status from multiple goroutines
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					_ = p.Status()
				}
				done <- true
			}()
		}

		// Wait for all goroutines to finish
		for i := 0; i < 10; i++ {
			select {
			case <-done:
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for concurrent status reads")
			}
		}
	})

	t.Run("concurrent action sends", func(t *testing.T) {
		p := New()
		ctx := context.Background()

		p.Run(ctx)
		defer p.Stop()

		actionCh := p.GetActionCh()

		// Send actions from multiple goroutines
		done := make(chan bool, 5)
		for i := 0; i < 5; i++ {
			go func() {
				select {
				case actionCh <- Inject:
				case <-time.After(100 * time.Millisecond):
				}
				done <- true
			}()
		}

		// Wait for all goroutines to finish
		for i := 0; i < 5; i++ {
			select {
			case <-done:
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for concurrent action sends")
			}
		}
	})
}

// Mock implementations for testing without external dependencies
type mockRecorder struct {
	started bool
	stopped bool
	frames  chan mockFrame
	errors  chan error
}

type mockFrame struct {
	data []byte
}

func (m *mockRecorder) Start(ctx context.Context) (<-chan mockFrame, <-chan error, error) {
	m.started = true
	m.frames = make(chan mockFrame, 10)
	m.errors = make(chan error, 1)

	// Send some mock frames
	go func() {
		defer close(m.frames)
		defer close(m.errors)

		for i := 0; i < 3; i++ {
			select {
			case m.frames <- mockFrame{data: []byte("mock audio data")}:
			case <-ctx.Done():
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	return m.frames, m.errors, nil
}

func (m *mockRecorder) Stop() error {
	m.stopped = true
	return nil
}

func TestPipelineWithMocks(t *testing.T) {
	t.Run("mock recorder lifecycle", func(t *testing.T) {
		recorder := &mockRecorder{}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		frames, errors, err := recorder.Start(ctx)
		if err != nil {
			t.Fatalf("mock recorder start failed: %v", err)
		}

		if !recorder.started {
			t.Error("recorder should be marked as started")
		}

		// Receive some frames
		frameCount := 0
		for frame := range frames {
			frameCount++
			if len(frame.data) == 0 {
				t.Error("frame should have data")
			}
		}

		if frameCount == 0 {
			t.Error("should receive some frames")
		}

		// Check for errors
		select {
		case err := <-errors:
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		default:
		}

		err = recorder.Stop()
		if err != nil {
			t.Errorf("mock recorder stop failed: %v", err)
		}

		if !recorder.stopped {
			t.Error("recorder should be marked as stopped")
		}
	})
}
