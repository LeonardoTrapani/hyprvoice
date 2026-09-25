package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/pipeline"
)

// newRecordTestDaemon builds a daemon against a throwaway config directory.
func newRecordTestDaemon(t *testing.T) *Daemon {
	t.Helper()

	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	configPath := filepath.Join(tempDir, "hyprvoice", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(testConfigContent), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(d.cancel)
	return d
}

// The press half must not disturb a recording already under way. With toggle
// on both edges, a duplicated press stops the recording the user is in the
// middle of; this is the guarantee that replaces that behaviour.
func TestStartRecordingIsIdempotent(t *testing.T) {
	d := newRecordTestDaemon(t)

	stub := &MockPipeline{status: pipeline.Transcribing}
	d.mu.Lock()
	d.pipeline = stub
	d.mu.Unlock()

	d.startRecording()

	d.mu.RLock()
	got := d.pipeline
	d.mu.RUnlock()

	if got != stub {
		t.Fatal("startRecording replaced a pipeline that was already recording")
	}
}

// The release half must be safe when there is nothing to release.
func TestStopRecordingWhenIdleDoesNothing(t *testing.T) {
	d := newRecordTestDaemon(t)

	// No pipeline at all: must not panic or leave one behind.
	d.stopRecording()

	d.mu.RLock()
	got := d.pipeline
	d.mu.RUnlock()

	if got != nil {
		t.Fatal("stopRecording created a pipeline while idle")
	}
}

func TestStopRecordingInjectsWhileTranscribing(t *testing.T) {
	d := newRecordTestDaemon(t)

	stub := &MockPipeline{status: pipeline.Transcribing, actionCh: make(chan pipeline.Action, 1)}
	d.mu.Lock()
	d.pipeline = stub
	d.mu.Unlock()

	d.stopRecording()

	select {
	case action := <-stub.actionCh:
		if action != pipeline.Inject {
			t.Fatalf("got action %q, want %q", action, pipeline.Inject)
		}
	case <-time.After(time.Second):
		t.Fatal("stopRecording sent no inject action")
	}
}

// A tap short enough to land inside the brief Recording state must still
// finish the dictation rather than leaving it running to the timeout. The
// buffered action channel is what makes that work.
func TestStopRecordingInjectsDuringTheRecordingWindow(t *testing.T) {
	d := newRecordTestDaemon(t)

	stub := &MockPipeline{status: pipeline.Recording, actionCh: make(chan pipeline.Action, 1)}
	d.mu.Lock()
	d.pipeline = stub
	d.mu.Unlock()

	d.stopRecording()

	select {
	case action := <-stub.actionCh:
		if action != pipeline.Inject {
			t.Fatalf("got action %q, want %q", action, pipeline.Inject)
		}
	case <-time.After(time.Second):
		t.Fatal("a stop during the Recording window was dropped")
	}
}

// Repeated releases must not queue up more work than the pipeline asked for.
func TestStopRecordingTwiceSendsOneAction(t *testing.T) {
	d := newRecordTestDaemon(t)

	stub := &MockPipeline{status: pipeline.Transcribing, actionCh: make(chan pipeline.Action, 1)}
	d.mu.Lock()
	d.pipeline = stub
	d.mu.Unlock()

	d.stopRecording()
	d.stopRecording() // must not block on the full buffer

	<-stub.actionCh
	select {
	case extra := <-stub.actionCh:
		t.Fatalf("a second action %q was queued", extra)
	default:
	}
}

// The sequence a lost release produces: press, (release dropped), press,
// release. The second cycle must still finish the dictation.
func TestLostReleaseRecoversOnTheNextCycle(t *testing.T) {
	d := newRecordTestDaemon(t)

	stub := &MockPipeline{status: pipeline.Transcribing, actionCh: make(chan pipeline.Action, 1)}
	d.mu.Lock()
	d.pipeline = stub
	d.mu.Unlock()

	// The release that never arrived is simply absent. The next press:
	d.startRecording()

	d.mu.RLock()
	same := d.pipeline == stub
	d.mu.RUnlock()
	if !same {
		t.Fatal("the press after a lost release restarted the pipeline")
	}

	// ...and the release after it still finishes the dictation.
	d.stopRecording()

	select {
	case action := <-stub.actionCh:
		if action != pipeline.Inject {
			t.Fatalf("got action %q, want %q", action, pipeline.Inject)
		}
	case <-time.After(time.Second):
		t.Fatal("the recording could not be stopped after a lost release")
	}
}

func TestStopRecordingIgnoresLateStates(t *testing.T) {
	for _, status := range []pipeline.Status{pipeline.Processing, pipeline.Injecting} {
		t.Run(string(status), func(t *testing.T) {
			d := newRecordTestDaemon(t)

			stub := &MockPipeline{status: status, actionCh: make(chan pipeline.Action, 1)}
			d.mu.Lock()
			d.pipeline = stub
			d.mu.Unlock()

			d.stopRecording()

			select {
			case action := <-stub.actionCh:
				t.Fatalf("sent %q while %s; the dictation was already finishing", action, status)
			default:
			}
		})
	}
}
