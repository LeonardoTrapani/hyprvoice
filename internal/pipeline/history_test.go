package pipeline

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/history"
	"github.com/leonardotrapani/hyprvoice/internal/testutil"
)

// historyTestConfig builds a pipeline config whose archive lands in a temp dir.
func historyTestConfig(t *testing.T, enabled bool, llm bool) *config.Config {
	t.Helper()
	return &config.Config{
		Recording: config.RecordingConfig{
			SampleRate: 16000, Channels: 1, Format: "s16",
			BufferSize: 1024, ChannelBufferSize: 10, Timeout: 5 * time.Second,
		},
		Transcription: config.TranscriptionConfig{
			Provider: "openai", Language: "en", Model: "whisper-1",
		},
		Injection: config.InjectionConfig{
			Backends: []string{"clipboard"}, ClipboardTimeout: 3 * time.Second,
		},
		Notifications: config.NotificationsConfig{Enabled: false, Type: "log"},
		LLM:           config.LLMConfig{Enabled: llm, Provider: "openai", Model: "gpt-4"},
		Providers:     map[string]config.ProviderConfig{"openai": {APIKey: "test-key"}},
		History: config.HistoryConfig{
			Enabled:    enabled,
			MaxEntries: 50,
			Path:       filepath.Join(t.TempDir(), "hyprvoice", "history.jsonl"),
		},
	}
}

// runToInjection drives a fully-mocked pipeline through one dictation.
func runToInjection(t *testing.T, cfg *config.Config, transcription string, injector *testutil.MockInjector, llmOut string) {
	t.Helper()

	opts := []Option{
		WithRecorderFactory(testutil.MockRecorderFactory(testutil.NewMockRecorder())),
		WithTranscriberFactory(testutil.MockTranscriberFactory(testutil.NewMockTranscriber(transcription))),
		WithInjectorFactory(testutil.MockInjectorFactory(injector)),
	}
	if cfg.LLM.Enabled {
		opts = append(opts, WithLLMAdapterFactory(testutil.MockLLMAdapterFactory(testutil.NewMockLLMAdapter(llmOut))))
	}

	p := New(cfg, opts...)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	p.Run(ctx)
	time.Sleep(50 * time.Millisecond)
	p.GetActionCh() <- Inject
	time.Sleep(150 * time.Millisecond)
}

func TestPipelineRecordsSuccessfulTranscription(t *testing.T) {
	cfg := historyTestConfig(t, true, false)
	runToInjection(t, cfg, "hello archive", testutil.NewMockInjector(), "")

	entries, err := history.New(cfg.History.Path, 50).List(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	e := entries[0]
	if e.Text != "hello archive" {
		t.Errorf("text %q", e.Text)
	}
	if !e.Injected {
		t.Error("Injected is false for a successful injection")
	}
	if e.Provider != "openai" || e.Model != "whisper-1" {
		t.Errorf("provider/model recorded as %q/%q", e.Provider, e.Model)
	}
}

// The case this feature exists for: injection failed, so the text must still
// be recoverable.
func TestPipelineRecordsFailedInjection(t *testing.T) {
	cfg := historyTestConfig(t, true, false)

	injector := testutil.NewMockInjector()
	injector.InjectError = context.DeadlineExceeded

	runToInjection(t, cfg, "text that never landed", injector, "")

	entries, err := history.New(cfg.History.Path, 50).List(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want the failed injection recorded", len(entries))
	}
	if entries[0].Injected {
		t.Error("Injected is true despite the injection failing")
	}
	if entries[0].Text != "text that never landed" {
		t.Errorf("text %q", entries[0].Text)
	}
}

// When post-processing rewrites the transcription, both versions are kept.
func TestPipelineRecordsRawTextWhenLLMRewrites(t *testing.T) {
	cfg := historyTestConfig(t, true, true)
	runToInjection(t, cfg, "um hello um world", testutil.NewMockInjector(), "Hello, World!")

	entries, err := history.New(cfg.History.Path, 50).List(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Text != "Hello, World!" {
		t.Errorf("text %q, want the processed version", entries[0].Text)
	}
	if entries[0].Raw != "um hello um world" {
		t.Errorf("raw %q, want the pre-LLM version", entries[0].Raw)
	}
}

func TestPipelineSkipsHistoryWhenDisabled(t *testing.T) {
	cfg := historyTestConfig(t, false, false)
	runToInjection(t, cfg, "should not be stored", testutil.NewMockInjector(), "")

	entries, err := history.New(cfg.History.Path, 50).List(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("got %d entries with history disabled, want 0", len(entries))
	}
}
