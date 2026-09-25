package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestStore(t *testing.T, maxEntries int) *Store {
	t.Helper()
	// Nested inside the temp dir so the store creates this directory itself,
	// which is what the permission test needs to observe.
	return New(filepath.Join(t.TempDir(), "hyprvoice", "history.jsonl"), maxEntries)
}

func TestAppendAndList(t *testing.T) {
	s := newTestStore(t, 10)

	for _, text := range []string{"first", "second", "third"} {
		if err := s.Append(Entry{Text: text, Injected: true}); err != nil {
			t.Fatalf("append %q: %v", text, err)
		}
	}

	got, err := s.List(0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d entries, want 3", len(got))
	}

	// Newest first.
	want := []string{"third", "second", "first"}
	for i := range want {
		if got[i].Text != want[i] {
			t.Fatalf("entry %d is %q, want %q", i, got[i].Text, want[i])
		}
	}
}

func TestListOnMissingFile(t *testing.T) {
	got, err := newTestStore(t, 10).List(0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d entries, want 0", len(got))
	}
}

func TestListRespectsLimit(t *testing.T) {
	s := newTestStore(t, 100)
	for i := 0; i < 10; i++ {
		if err := s.Append(Entry{Text: string(rune('a' + i))}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.List(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d entries, want 3", len(got))
	}
	if got[0].Text != "j" {
		t.Fatalf("newest is %q, want %q", got[0].Text, "j")
	}
}

// A recording that transcribed to nothing is not worth an entry.
func TestAppendIgnoresEmptyText(t *testing.T) {
	s := newTestStore(t, 10)
	for _, text := range []string{"", "   ", "\n\t "} {
		if err := s.Append(Entry{Text: text}); err != nil {
			t.Fatalf("append %q: %v", text, err)
		}
	}

	got, _ := s.List(0)
	if len(got) != 0 {
		t.Fatalf("got %d entries, want 0", len(got))
	}
}

func TestAppendAssignsIDAndTimestamp(t *testing.T) {
	s := newTestStore(t, 10)
	if err := s.Append(Entry{Text: "hello"}); err != nil {
		t.Fatal(err)
	}

	got, _ := s.List(0)
	if got[0].ID == "" {
		t.Error("ID was not assigned")
	}
	if got[0].At.IsZero() {
		t.Error("timestamp was not assigned")
	}
}

// Raw is only interesting when post-processing changed something.
func TestAppendDropsRawWhenIdentical(t *testing.T) {
	s := newTestStore(t, 10)
	if err := s.Append(Entry{Text: "same", Raw: "same"}); err != nil {
		t.Fatal(err)
	}

	got, _ := s.List(0)
	if got[0].Raw != "" {
		t.Fatalf("Raw is %q, want it dropped as identical", got[0].Raw)
	}
}

func TestAppendKeepsRawWhenDifferent(t *testing.T) {
	s := newTestStore(t, 10)
	if err := s.Append(Entry{Text: "Hello, world!", Raw: "um hello world"}); err != nil {
		t.Fatal(err)
	}

	got, _ := s.List(0)
	if got[0].Raw != "um hello world" {
		t.Fatalf("Raw is %q, want the pre-LLM text", got[0].Raw)
	}
}

func TestGet(t *testing.T) {
	s := newTestStore(t, 10)
	if err := s.Append(Entry{Text: "findme"}); err != nil {
		t.Fatal(err)
	}

	all, _ := s.List(0)
	got, ok, err := s.Get(all[0].ID)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.Text != "findme" {
		t.Fatalf("got %q", got.Text)
	}

	if _, ok, _ := s.Get("nope"); ok {
		t.Error("found an entry that does not exist")
	}
}

func TestLast(t *testing.T) {
	s := newTestStore(t, 10)

	if _, ok, err := s.Last(); ok || err != nil {
		t.Fatalf("empty store: ok=%v err=%v", ok, err)
	}

	for _, text := range []string{"old", "new"} {
		if err := s.Append(Entry{Text: text}); err != nil {
			t.Fatal(err)
		}
	}

	got, ok, err := s.Last()
	if err != nil || !ok {
		t.Fatalf("last: ok=%v err=%v", ok, err)
	}
	if got.Text != "new" {
		t.Fatalf("got %q, want %q", got.Text, "new")
	}
}

func TestClear(t *testing.T) {
	s := newTestStore(t, 10)
	if err := s.Append(Entry{Text: "gone"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}

	got, _ := s.List(0)
	if len(got) != 0 {
		t.Fatalf("got %d entries after clear, want 0", len(got))
	}

	// Clearing an already-empty store is not an error.
	if err := s.Clear(); err != nil {
		t.Fatalf("second clear: %v", err)
	}
}

// The file may exceed the limit between compactions; what matters is that it
// is bounded and that the newest entries are the survivors.
func TestCompactionBoundsTheFile(t *testing.T) {
	const maxEntries = 5
	s := newTestStore(t, maxEntries)

	for i := 0; i < 100; i++ {
		if err := s.Append(Entry{Text: string(rune('a' + i%26)), Injected: true}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.List(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) > maxEntries*compactionFactor {
		t.Fatalf("store holds %d entries, want at most %d", len(got), maxEntries*compactionFactor)
	}

	// The newest write must always survive.
	if got[0].Text != string(rune('a'+99%26)) {
		t.Fatalf("newest entry is %q, want %q", got[0].Text, string(rune('a'+99%26)))
	}
}

func TestCompactionKeepsNewest(t *testing.T) {
	s := newTestStore(t, 3)
	for i := 0; i < 20; i++ {
		if err := s.Append(Entry{Text: string(rune('a' + i))}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.compact(); err != nil {
		t.Fatal(err)
	}

	got, _ := s.List(0)
	if len(got) != 3 {
		t.Fatalf("got %d entries after compact, want 3", len(got))
	}
	if got[0].Text != "t" {
		t.Fatalf("newest after compact is %q, want %q", got[0].Text, "t")
	}
}

// A torn write should cost that one entry, not the archive.
func TestReadSkipsCorruptLines(t *testing.T) {
	s := newTestStore(t, 10)
	if err := s.Append(Entry{Text: "good one"}); err != nil {
		t.Fatal(err)
	}

	f, err := os.OpenFile(s.Path(), os.O_APPEND|os.O_WRONLY, filePerm)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("{ this is not json\n\n")
	f.Close()

	if err := s.Append(Entry{Text: "good two"}); err != nil {
		t.Fatal(err)
	}

	got, err := s.List(0)
	if err != nil {
		t.Fatalf("a corrupt line broke the whole read: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want the 2 valid ones", len(got))
	}
}

// Transcripts are as sensitive as the clipboard; they must not be world
// readable.
func TestFilePermissionsArePrivate(t *testing.T) {
	s := newTestStore(t, 10)
	if err := s.Append(Entry{Text: "secret"}); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != filePerm {
		t.Fatalf("history file is %o, want %o", perm, filePerm)
	}

	di, err := os.Stat(filepath.Dir(s.Path()))
	if err != nil {
		t.Fatal(err)
	}
	if perm := di.Mode().Perm(); perm != dirPerm {
		t.Fatalf("history directory is %o, want %o", perm, dirPerm)
	}
}

func TestCompactionPreservesPermissions(t *testing.T) {
	s := newTestStore(t, 2)
	for i := 0; i < 30; i++ {
		if err := s.Append(Entry{Text: string(rune('a' + i%26))}); err != nil {
			t.Fatal(err)
		}
	}

	fi, err := os.Stat(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != filePerm {
		t.Fatalf("after compaction the file is %o, want %o", perm, filePerm)
	}
}

// A long dictation must round-trip: the scanner's default token limit would
// reject it.
func TestLongTranscriptionRoundTrips(t *testing.T) {
	s := newTestStore(t, 10)
	long := strings.Repeat("the quick brown fox jumps over the lazy dog. ", 5000)

	if err := s.Append(Entry{Text: long}); err != nil {
		t.Fatal(err)
	}

	got, err := s.List(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Text != long {
		t.Fatalf("long transcription did not round-trip (got %d entries)", len(got))
	}
}

func TestUnicodeRoundTrips(t *testing.T) {
	s := newTestStore(t, 10)
	text := "café — naïve “quotes” 日本語 🎙"

	if err := s.Append(Entry{Text: text}); err != nil {
		t.Fatal(err)
	}

	got, _ := s.List(0)
	if got[0].Text != text {
		t.Fatalf("got %q, want %q", got[0].Text, text)
	}
}

// Newlines are sanitised out before injection, but the store holds whatever it
// is given and must not let one entry become two.
func TestNewlinesDoNotSplitEntries(t *testing.T) {
	s := newTestStore(t, 10)
	if err := s.Append(Entry{Text: "line one\nline two\nline three"}); err != nil {
		t.Fatal(err)
	}

	got, _ := s.List(0)
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1", len(got))
	}
	if got[0].Text != "line one\nline two\nline three" {
		t.Fatalf("got %q", got[0].Text)
	}
}

func TestEntryJSONShape(t *testing.T) {
	at := time.Date(2026, 9, 14, 10, 30, 0, 0, time.UTC)
	line, err := json.Marshal(Entry{ID: "x", At: at, Text: "hi", Injected: true})
	if err != nil {
		t.Fatal(err)
	}

	// Omitted when empty, so the common entry stays compact.
	for _, absent := range []string{"raw", "provider", "model"} {
		if strings.Contains(string(line), `"`+absent+`"`) {
			t.Errorf("empty %q should have been omitted: %s", absent, line)
		}
	}
	for _, present := range []string{"id", "at", "text", "injected"} {
		if !strings.Contains(string(line), `"`+present+`"`) {
			t.Errorf("%q missing from %s", present, line)
		}
	}
}

func TestNewClampsInvalidMaxEntries(t *testing.T) {
	for _, n := range []int{0, -1} {
		if got := New("/tmp/x", n).maxEntries; got != DefaultMaxEntries {
			t.Fatalf("New(%d) gave maxEntries %d, want %d", n, got, DefaultMaxEntries)
		}
	}
}

func TestDefaultPathUsesXDGState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/custom/state")

	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/custom/state/hyprvoice/history.jsonl" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultPathFallsBackToHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")

	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, filepath.Join(".local", "state", "hyprvoice", "history.jsonl")) {
		t.Fatalf("got %q", got)
	}
}

// Appends happen on the pipeline goroutine while a picker may be reading.
func TestConcurrentAppends(t *testing.T) {
	s := newTestStore(t, 1000)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if err := s.Append(Entry{Text: "entry"}); err != nil {
					t.Errorf("append: %v", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	got, err := s.List(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 160 {
		t.Fatalf("got %d entries, want 160", len(got))
	}
}
