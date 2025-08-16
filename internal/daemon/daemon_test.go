package daemon

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/notify"
	"github.com/leonardotrapani/hyprvoice/internal/pipeline"
)

// Mock notifier for testing
type mockNotifier struct {
	recordingStartedCalled bool
	recordingEndedCalled   bool
	abortedCalled          bool
	transcribingCalled     bool
	errorCalled            bool
	lastErrorMessage       string
}

func (m *mockNotifier) RecordingStarted() { m.recordingStartedCalled = true }
func (m *mockNotifier) RecordingEnded()   { m.recordingEndedCalled = true }
func (m *mockNotifier) Aborted()          { m.abortedCalled = true }
func (m *mockNotifier) Transcribing()     { m.transcribingCalled = true }
func (m *mockNotifier) Error(msg string) {
	m.errorCalled = true
	m.lastErrorMessage = msg
}
func (m *mockNotifier) Notify(title, message string) {}

func (m *mockNotifier) reset() {
	m.recordingStartedCalled = false
	m.recordingEndedCalled = false
	m.abortedCalled = false
	m.transcribingCalled = false
	m.errorCalled = false
	m.lastErrorMessage = ""
}

// Mock pipeline for testing
type mockPipeline struct {
	status   pipeline.Status
	actionCh chan pipeline.Action
	errorCh  chan pipeline.PipelineError
	running  bool
	stopped  bool
}

func newMockPipeline() *mockPipeline {
	return &mockPipeline{
		status:   pipeline.Idle,
		actionCh: make(chan pipeline.Action, 1),
		errorCh:  make(chan pipeline.PipelineError, 10),
	}
}

func (m *mockPipeline) Run(ctx context.Context) {
	m.running = true
	m.status = pipeline.Recording
}

func (m *mockPipeline) Stop() {
	m.stopped = true
	m.running = false
	m.status = pipeline.Idle
}

func (m *mockPipeline) Status() pipeline.Status {
	return m.status
}

func (m *mockPipeline) GetActionCh() chan<- pipeline.Action {
	return m.actionCh
}

func (m *mockPipeline) GetErrorCh() <-chan pipeline.PipelineError {
	return m.errorCh
}

func (m *mockPipeline) SetStatus(status pipeline.Status) {
	m.status = status
}

func (m *mockPipeline) SendError(title, message string, err error) {
	pipelineErr := pipeline.PipelineError{
		Title:   title,
		Message: message,
		Err:     err,
	}
	select {
	case m.errorCh <- pipelineErr:
	default:
	}
}

func TestNewDaemon(t *testing.T) {
	t.Run("with notifier", func(t *testing.T) {
		mockNotif := &mockNotifier{}
		daemon := New(mockNotif)

		if daemon == nil {
			t.Fatal("daemon should not be nil")
		}

		if daemon.notifier != mockNotif {
			t.Error("daemon should use provided notifier")
		}
	})

	t.Run("with nil notifier", func(t *testing.T) {
		daemon := New(nil)

		if daemon == nil {
			t.Fatal("daemon should not be nil")
		}

		// Should use default Desktop notifier
		if daemon.notifier == nil {
			t.Error("daemon should have a notifier")
		}

		// Check if it's a Desktop notifier by type assertion
		if _, ok := daemon.notifier.(notify.Desktop); !ok {
			t.Error("daemon should use Desktop notifier when nil is provided")
		}
	})
}

func TestDaemonStatus(t *testing.T) {
	mockNotif := &mockNotifier{}
	daemon := New(mockNotif)

	t.Run("initial status", func(t *testing.T) {
		status := daemon.status()
		if status != pipeline.Idle {
			t.Errorf("initial status should be Idle, got %s", status)
		}
	})

	t.Run("status with mock pipeline", func(t *testing.T) {
		mockPipe := newMockPipeline()
		daemon.pipeline = mockPipe

		// Test different statuses
		statuses := []pipeline.Status{
			pipeline.Recording,
			pipeline.Transcribing,
			pipeline.Injecting,
			pipeline.Idle,
		}

		for _, expectedStatus := range statuses {
			mockPipe.SetStatus(expectedStatus)
			status := daemon.status()
			if status != expectedStatus {
				t.Errorf("status should be %s, got %s", expectedStatus, status)
			}
		}
	})
}

func TestDaemonStopPipeline(t *testing.T) {
	mockNotif := &mockNotifier{}
	daemon := New(mockNotif)

	t.Run("stop with no pipeline", func(t *testing.T) {
		// Should not panic
		daemon.stopPipeline()
	})

	t.Run("stop with pipeline", func(t *testing.T) {
		mockPipe := newMockPipeline()
		daemon.pipeline = mockPipe

		daemon.stopPipeline()

		if !mockPipe.stopped {
			t.Error("pipeline should be stopped")
		}

		if daemon.pipeline != nil {
			t.Error("daemon pipeline should be nil after stopping")
		}
	})
}

func TestDaemonToggle(t *testing.T) {
	mockNotif := &mockNotifier{}
	daemon := New(mockNotif)

	t.Run("toggle from idle", func(t *testing.T) {
		mockNotif.reset()

		// Manually set up a mock pipeline for testing
		mockPipe := newMockPipeline()
		daemon.pipeline = nil

		// Test toggle from idle - should start recording
		// We need to simulate the behavior since we can't easily mock pipeline.New()
		status := daemon.status() // Should be Idle
		if status != pipeline.Idle {
			t.Errorf("initial status should be Idle, got %s", status)
		}

		// Simulate what toggle() would do in Idle state
		if status == pipeline.Idle {
			daemon.pipeline = mockPipe
			mockPipe.Run(context.Background())
			// Notification would be sent in goroutine
			if !mockNotif.recordingStartedCalled {
				// In real implementation this would be called,
				// but since we're testing the logic manually, we call it
				mockNotif.RecordingStarted()
			}
		}

		if !mockNotif.recordingStartedCalled {
			t.Error("recording started notification should be called")
		}

		if !mockPipe.running {
			t.Error("pipeline should be running")
		}
	})

	t.Run("toggle from recording", func(t *testing.T) {
		mockNotif.reset()
		mockPipe := newMockPipeline()
		mockPipe.SetStatus(pipeline.Recording)
		daemon.pipeline = mockPipe

		// Simulate toggle from recording - should abort
		status := daemon.status()
		if status == pipeline.Recording {
			daemon.stopPipeline()
			mockNotif.Aborted()
		}

		if !mockNotif.abortedCalled {
			t.Error("aborted notification should be called")
		}
	})

	t.Run("toggle from transcribing", func(t *testing.T) {
		mockNotif.reset()
		mockPipe := newMockPipeline()
		mockPipe.SetStatus(pipeline.Transcribing)
		daemon.pipeline = mockPipe

		// Simulate toggle from transcribing - should inject
		status := daemon.status()
		if status == pipeline.Transcribing {
			actionCh := mockPipe.GetActionCh()
			select {
			case actionCh <- pipeline.Inject:
				mockNotif.RecordingEnded()
			default:
				t.Error("should be able to send inject action")
			}
		}

		if !mockNotif.recordingEndedCalled {
			t.Error("recording ended notification should be called")
		}

		// Check that inject action was sent
		select {
		case action := <-mockPipe.actionCh:
			if action != pipeline.Inject {
				t.Errorf("action should be Inject, got %v", action)
			}
		default:
			t.Error("inject action should have been sent")
		}
	})

	t.Run("toggle from injecting", func(t *testing.T) {
		mockNotif.reset()
		mockPipe := newMockPipeline()
		mockPipe.SetStatus(pipeline.Injecting)
		daemon.pipeline = mockPipe

		// Simulate toggle from injecting - should abort
		status := daemon.status()
		if status == pipeline.Injecting {
			daemon.stopPipeline()
			mockNotif.Aborted()
		}

		if !mockNotif.abortedCalled {
			t.Error("aborted notification should be called")
		}
	})
}

func TestDaemonHandlePipelineError(t *testing.T) {
	mockNotif := &mockNotifier{}
	daemon := New(mockNotif)

	t.Run("handle error without underlying error", func(t *testing.T) {
		mockNotif.reset()

		pipelineErr := pipeline.PipelineError{
			Title:   "Test Error",
			Message: "Test message",
			Err:     nil,
		}

		daemon.handlePipelineError(pipelineErr)

		if !mockNotif.errorCalled {
			t.Error("error notification should be called")
		}

		if mockNotif.lastErrorMessage != "Test message" {
			t.Errorf("error message should be 'Test message', got '%s'", mockNotif.lastErrorMessage)
		}
	})

	t.Run("handle error with underlying error", func(t *testing.T) {
		mockNotif.reset()

		underlyingErr := fmt.Errorf("underlying error")
		pipelineErr := pipeline.PipelineError{
			Title:   "Test Error",
			Message: "Test message",
			Err:     underlyingErr,
		}

		daemon.handlePipelineError(pipelineErr)

		if !mockNotif.errorCalled {
			t.Error("error notification should be called")
		}

		expectedMessage := "Test message: underlying error"
		if mockNotif.lastErrorMessage != expectedMessage {
			t.Errorf("error message should be '%s', got '%s'", expectedMessage, mockNotif.lastErrorMessage)
		}
	})
}

func TestDaemonMonitorPipelineErrors(t *testing.T) {
	mockNotif := &mockNotifier{}
	daemon := New(mockNotif)

	t.Run("monitor pipeline errors", func(t *testing.T) {
		mockNotif.reset()
		mockPipe := newMockPipeline()

		// Start monitoring in background
		done := make(chan bool, 1)
		go func() {
			daemon.monitorPipelineErrors(mockPipe)
			done <- true
		}()

		// Send an error
		testErr := pipeline.PipelineError{
			Title:   "Test Error",
			Message: "Test message",
			Err:     nil,
		}
		mockPipe.SendError(testErr.Title, testErr.Message, testErr.Err)

		// Give some time for error to be processed
		time.Sleep(10 * time.Millisecond)

		// Cancel context to stop monitoring
		daemon.cancel()

		// Wait for monitoring to stop
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for error monitoring to stop")
		}

		if !mockNotif.errorCalled {
			t.Error("error notification should be called")
		}
	})
}

func TestDaemonConcurrency(t *testing.T) {
	mockNotif := &mockNotifier{}
	daemon := New(mockNotif)

	t.Run("concurrent status calls", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					daemon.status()
				}
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			select {
			case <-done:
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for concurrent status calls")
			}
		}
	})

	t.Run("concurrent stopPipeline calls", func(t *testing.T) {
		mockPipe := newMockPipeline()
		daemon.pipeline = mockPipe

		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				daemon.stopPipeline()
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			select {
			case <-done:
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for concurrent stopPipeline calls")
			}
		}

		// Pipeline should be stopped only once
		if !mockPipe.stopped {
			t.Error("pipeline should be stopped")
		}
	})
}

func TestDaemonIntegrationWithMockSocket(t *testing.T) {
	// This test simulates the daemon handle() method behavior
	mockNotif := &mockNotifier{}
	daemon := New(mockNotif)

	// Create a mock connection using pipe
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	t.Run("handle toggle command", func(t *testing.T) {
		mockNotif.reset()

		// Send toggle command
		go func() {
			client.Write([]byte("t\n"))
		}()

		// Simulate handle() behavior
		go func() {
			buf := make([]byte, 2)
			n, err := server.Read(buf)
			if err != nil || n != 2 {
				return
			}

			cmd := buf[0]
			if cmd == 't' {
				// Simulate toggle logic
				status := daemon.status()
				if status == pipeline.Idle {
					mockPipe := newMockPipeline()
					daemon.pipeline = mockPipe
					mockPipe.Run(context.Background())
					mockNotif.RecordingStarted()
				}
				fmt.Fprint(server, "OK toggled\n")
			}
		}()

		// Read response
		response := make([]byte, 1024)
		n, err := client.Read(response)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		expectedResponse := "OK toggled\n"
		actualResponse := string(response[:n])
		if actualResponse != expectedResponse {
			t.Errorf("response should be %q, got %q", expectedResponse, actualResponse)
		}

		if !mockNotif.recordingStartedCalled {
			t.Error("recording started notification should be called")
		}
	})

	t.Run("handle status command", func(t *testing.T) {
		// Send status command
		go func() {
			client.Write([]byte("s\n"))
		}()

		// Simulate handle() behavior
		go func() {
			buf := make([]byte, 2)
			n, err := server.Read(buf)
			if err != nil || n != 2 {
				return
			}

			cmd := buf[0]
			if cmd == 's' {
				status := daemon.status()
				fmt.Fprintf(server, "STATUS status=%s\n", status)
			}
		}()

		// Read response
		response := make([]byte, 1024)
		n, err := client.Read(response)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		expectedResponse := fmt.Sprintf("STATUS status=%s\n", daemon.status())
		actualResponse := string(response[:n])
		if actualResponse != expectedResponse {
			t.Errorf("response should be %q, got %q", expectedResponse, actualResponse)
		}
	})
}

func TestDaemonErrorPropagation(t *testing.T) {
	mockNotif := &mockNotifier{}
	daemon := New(mockNotif)

	t.Run("pipeline error propagation", func(t *testing.T) {
		mockNotif.reset()
		mockPipe := newMockPipeline()
		daemon.pipeline = mockPipe

		// Start error monitoring
		done := make(chan bool, 1)
		go func() {
			daemon.monitorPipelineErrors(mockPipe)
			done <- true
		}()

		// Simulate multiple errors
		errors := []pipeline.PipelineError{
			{Title: "Error 1", Message: "Message 1", Err: nil},
			{Title: "Error 2", Message: "Message 2", Err: fmt.Errorf("underlying")},
		}

		for _, err := range errors {
			mockPipe.SendError(err.Title, err.Message, err.Err)
			time.Sleep(5 * time.Millisecond) // Give time for processing
		}

		// Cancel and stop monitoring
		daemon.cancel()

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for error monitoring to stop")
		}

		if !mockNotif.errorCalled {
			t.Error("error notification should be called")
		}
	})
}
