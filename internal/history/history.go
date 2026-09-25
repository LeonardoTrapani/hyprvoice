// Package history keeps a rolling record of what hyprvoice transcribed.
//
// Dictation is fire-and-forget: text is typed into whichever window had focus
// and is gone the moment that window did not want it -- the wrong window was
// focused, the app swallowed it, injection failed outright. The transcription
// itself was the expensive part, so it is worth keeping.
//
// The store is a JSONL file. Appending a line is one open-and-write, so the
// cost sits off the dictation path; the file is compacted back down to the
// entry limit only once it has drifted well past it.
package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// DefaultMaxEntries is how many transcriptions are kept.
	DefaultMaxEntries = 200

	// compactionFactor sets how far past the limit the file may grow before
	// it is rewritten. Compacting on every append would turn each dictation
	// into a full read-modify-write of the whole store.
	compactionFactor = 2

	dirPerm  = 0o700
	filePerm = 0o600
)

// Entry is one transcription.
type Entry struct {
	ID string    `json:"id"`
	At time.Time `json:"at"`

	// Text is what was actually injected.
	Text string `json:"text"`

	// Raw is the transcription before LLM post-processing, recorded only when
	// post-processing changed it. It is what you want when the cleanup pass
	// rewrote something it should have left alone.
	Raw string `json:"raw,omitempty"`

	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`

	// Injected reports whether the text reached a window. A false here is the
	// case this package exists for.
	Injected bool `json:"injected"`
}

// Store is an append-only transcription log.
type Store struct {
	path       string
	maxEntries int
}

// New opens the store at path. The file is created on first append.
func New(path string, maxEntries int) *Store {
	if maxEntries <= 0 {
		maxEntries = DefaultMaxEntries
	}
	return &Store{path: path, maxEntries: maxEntries}
}

// DefaultPath is the store's location under the XDG state directory, which is
// where data that is useful to keep but not worth backing up belongs.
func DefaultPath() (string, error) {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "hyprvoice", "history.jsonl"), nil
}

// Path returns the file backing the store.
func (s *Store) Path() string { return s.path }

// Append records a transcription. Entries with no text are ignored: a
// recording that produced nothing is not worth keeping.
func (s *Store) Append(e Entry) error {
	if strings.TrimSpace(e.Text) == "" {
		return nil
	}

	if e.At.IsZero() {
		e.At = time.Now()
	}
	if e.ID == "" {
		e.ID = newID(e.At)
	}

	// Storing Raw only when it differs keeps the common case to one copy of
	// the text rather than two.
	if e.Raw == e.Text {
		e.Raw = ""
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create history directory: %w", err)
	}

	// MkdirAll's mode is masked by the process umask, which on a default Arch
	// setup yields 0755. These are transcripts of everything the user has
	// dictated, so set the mode explicitly rather than inheriting it.
	if err := os.Chmod(dir, dirPerm); err != nil {
		return fmt.Errorf("secure history directory: %w", err)
	}

	line, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("encode history entry: %w", err)
	}

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return fmt.Errorf("open history: %w", err)
	}

	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return fmt.Errorf("write history: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close history: %w", err)
	}

	return s.compactIfNeeded()
}

// List returns entries newest first, at most limit of them. A limit of zero or
// less returns everything kept.
func (s *Store) List(limit int) ([]Entry, error) {
	entries, err := s.readAll()
	if err != nil {
		return nil, err
	}

	// Stored oldest-first by append order; callers want the most recent.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

// Last returns the most recent entry. Returns false if the store is empty.
func (s *Store) Last() (Entry, bool, error) {
	entries, err := s.List(1)
	if err != nil || len(entries) == 0 {
		return Entry{}, false, err
	}
	return entries[0], true, nil
}

// Get returns the entry with the given ID.
func (s *Store) Get(id string) (Entry, bool, error) {
	entries, err := s.readAll()
	if err != nil {
		return Entry{}, false, err
	}
	for _, e := range entries {
		if e.ID == id {
			return e, true, nil
		}
	}
	return Entry{}, false, nil
}

// Clear removes every entry.
func (s *Store) Clear() error {
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear history: %w", err)
	}
	return nil
}

// readAll parses the store oldest-first.
//
// A malformed line is skipped rather than failing the read: a torn write from
// a killed daemon should cost that one entry, not the whole archive.
func (s *Store) readAll() ([]Entry, error) {
	f, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open history: %w", err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)

	// Transcriptions can be long; the default 64KiB token limit would reject
	// a genuinely large dictation.
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read history: %w", err)
	}
	return entries, nil
}

// compactIfNeeded rewrites the file down to maxEntries once it has grown to
// compactionFactor times that, so appends stay O(1) amortised.
func (s *Store) compactIfNeeded() error {
	n, err := s.countLines()
	if err != nil || n <= s.maxEntries*compactionFactor {
		return err
	}
	return s.compact()
}

func (s *Store) countLines() (int, error) {
	f, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var n int
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		n++
	}
	return n, scanner.Err()
}

// compact writes the newest maxEntries to a temporary file and renames it over
// the original, so a crash mid-compaction leaves the previous store intact.
func (s *Store) compact() error {
	entries, err := s.readAll()
	if err != nil {
		return err
	}
	if len(entries) > s.maxEntries {
		entries = entries[len(entries)-s.maxEntries:]
	}

	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, "history-*.jsonl")
	if err != nil {
		return fmt.Errorf("create temp history: %w", err)
	}
	tmpName := tmp.Name()

	if err := tmp.Chmod(filePerm); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("chmod temp history: %w", err)
	}

	w := bufio.NewWriter(tmp)
	for _, e := range entries {
		line, err := json.Marshal(e)
		if err != nil {
			continue
		}
		if _, err := w.Write(append(line, '\n')); err != nil {
			tmp.Close()
			os.Remove(tmpName)
			return fmt.Errorf("write temp history: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("flush temp history: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp history: %w", err)
	}

	if err := os.Rename(tmpName, s.path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("replace history: %w", err)
	}
	return nil
}

// newID is a sortable, human-legible identifier. Transcriptions arrive far
// apart in machine terms, so the timestamp alone distinguishes them, and the
// nanosecond component breaks any tie.
func newID(at time.Time) string {
	return fmt.Sprintf("%s-%03d", at.Format("20060102T150405"), at.Nanosecond()/1e6)
}
