package injection

import (
	"strings"
	"testing"
	"time"
)

func TestWtypeArgsWithoutDelays(t *testing.T) {
	w := &wtypeBackend{}

	got := w.args("hello")
	want := []string{"--", "hello"}

	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestWtypeArgsWithDelays(t *testing.T) {
	w := &wtypeBackend{startDelay: 80 * time.Millisecond, keyDelay: 4 * time.Millisecond}

	got := strings.Join(w.args("hello"), " ")
	want := "-s 80 -d 4 -- hello"

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// "--" must stay immediately before the text so a transcription starting with
// a dash is never read as a flag.
func TestWtypeArgsTerminatesOptions(t *testing.T) {
	w := &wtypeBackend{startDelay: 50 * time.Millisecond}

	got := w.args("-not-a-flag")
	if got[len(got)-2] != "--" {
		t.Fatalf("args %v do not terminate options before the text", got)
	}
	if got[len(got)-1] != "-not-a-flag" {
		t.Fatalf("text was altered: %q", got[len(got)-1])
	}
}

// Sub-millisecond delays round to zero and must not produce a bare "-s 0".
func TestWtypeArgsOmitsZeroDelays(t *testing.T) {
	w := &wtypeBackend{startDelay: 100 * time.Microsecond}

	for _, arg := range w.args("hello") {
		if arg == "-s" || arg == "-d" {
			t.Fatalf("emitted %q for a sub-millisecond delay: %v", arg, w.args("hello"))
		}
	}
}
