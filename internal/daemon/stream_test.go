package daemon

import (
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/pipeline"
)

func TestHubBroadcastReachesEverySubscriber(t *testing.T) {
	h := newStreamHub()
	a, b := h.add(), h.add()

	h.broadcast(pipeline.StatusEvent{Status: pipeline.Recording, Listening: true})

	for name, sub := range map[string]*subscriber{"a": a, "b": b} {
		select {
		case ev := <-sub.ch:
			if ev.Status != pipeline.Recording || !ev.Listening {
				t.Fatalf("%s: got %+v", name, ev)
			}
		default:
			t.Fatalf("%s received nothing", name)
		}
	}
}

func TestHubRemoveStopsDelivery(t *testing.T) {
	h := newStreamHub()
	s := h.add()
	h.remove(s)

	h.broadcast(pipeline.StatusEvent{Status: pipeline.Recording})

	select {
	case ev := <-s.ch:
		t.Fatalf("removed subscriber still got %+v", ev)
	default:
	}
}

func TestHubRemoveIsIdempotent(t *testing.T) {
	h := newStreamHub()
	s := h.add()

	h.remove(s)
	h.remove(s) // must not panic, nor drive the count negative

	if got := h.count(); got != 0 {
		t.Fatalf("count %d, want 0", got)
	}
}

// The audio path must never be held up by something that stopped reading.
func TestHubBroadcastDoesNotBlockOnSlowSubscriber(t *testing.T) {
	h := newStreamHub()
	h.add() // never drained

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < subscriberBuffer*4; i++ {
			h.broadcast(pipeline.StatusEvent{Status: pipeline.Recording})
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("broadcast blocked on a subscriber that stopped reading")
	}
}

func TestHubOnCountTracksSubscribers(t *testing.T) {
	h := newStreamHub()

	counts := make(chan int, 8)
	h.setOnCount(func(n int) { counts <- n })

	a := h.add()
	b := h.add()
	h.remove(a)
	h.remove(b)

	var got []int
	for i := 0; i < 4; i++ {
		select {
		case n := <-counts:
			got = append(got, n)
		case <-time.After(time.Second):
			t.Fatalf("only got %v", got)
		}
	}

	want := []int{1, 2, 1, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("counts %v, want %v", got, want)
		}
	}
}

func TestHubCount(t *testing.T) {
	h := newStreamHub()
	if got := h.count(); got != 0 {
		t.Fatalf("fresh hub count %d, want 0", got)
	}

	s := h.add()
	if got := h.count(); got != 1 {
		t.Fatalf("count %d, want 1", got)
	}

	h.remove(s)
	if got := h.count(); got != 0 {
		t.Fatalf("count %d, want 0", got)
	}
}

// Concurrent subscribe/unsubscribe against a steady broadcast is the normal
// shape of this code in production, and is what -race is here to check.
func TestHubConcurrentUse(t *testing.T) {
	h := newStreamHub()

	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				h.broadcast(pipeline.StatusEvent{Status: pipeline.Recording})
			}
		}
	}()

	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 50; j++ {
				s := h.add()
				select {
				case <-s.ch:
				default:
				}
				h.remove(s)
			}
		}()
	}

	for i := 0; i < 8; i++ {
		<-done
	}
	close(stop)

	if got := h.count(); got != 0 {
		t.Fatalf("count %d after all subscribers left, want 0", got)
	}
}
