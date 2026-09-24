package daemon

import (
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/pipeline"
)

// subscriberBuffer is how many events a single slow subscriber may fall behind
// before it starts missing them. Levels arrive in bursts of one per frame, so
// this is roughly a second of animation at the default buffer size.
const subscriberBuffer = 32

type subscriber struct {
	ch chan pipeline.StatusEvent
}

// streamHub fans pipeline events out to everything watching the status stream.
//
// Sends never block: a subscriber that stops reading loses events rather than
// stalling the pipeline that produced them. Losing a level event costs a frame
// of animation, and a dropped status change is corrected by the next one.
type streamHub struct {
	mu   sync.RWMutex
	subs map[*subscriber]struct{}

	// onCount runs whenever the subscriber count changes, so the daemon can
	// switch loudness measurement off while nothing is watching.
	onCount func(int)
}

func newStreamHub() *streamHub {
	return &streamHub{subs: make(map[*subscriber]struct{})}
}

func (h *streamHub) setOnCount(fn func(int)) {
	h.mu.Lock()
	h.onCount = fn
	h.mu.Unlock()
}

func (h *streamHub) add() *subscriber {
	s := &subscriber{ch: make(chan pipeline.StatusEvent, subscriberBuffer)}

	h.mu.Lock()
	h.subs[s] = struct{}{}
	n, fn := len(h.subs), h.onCount
	h.mu.Unlock()

	if fn != nil {
		fn(n)
	}
	return s
}

func (h *streamHub) remove(s *subscriber) {
	h.mu.Lock()
	if _, ok := h.subs[s]; !ok {
		h.mu.Unlock()
		return
	}
	delete(h.subs, s)
	n, fn := len(h.subs), h.onCount
	h.mu.Unlock()

	if fn != nil {
		fn(n)
	}
}

func (h *streamHub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}

func (h *streamHub) broadcast(ev pipeline.StatusEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for s := range h.subs {
		select {
		case s.ch <- ev:
		default:
		}
	}
}

// monitorPipelineEvents forwards one pipeline's events to every subscriber
// until the daemon shuts down or the pipeline's channel closes.
func (d *Daemon) monitorPipelineEvents(p pipeline.Pipeline) {
	eventCh := p.GetEventCh()
	for {
		select {
		case ev, ok := <-eventCh:
			if !ok {
				return
			}
			d.hub.broadcast(ev)
		case <-d.ctx.Done():
			return
		}
	}
}

// setLevelsWanted tells the running pipeline, if any, whether to measure
// loudness. Called on every subscriber count change.
func (d *Daemon) setLevelsWanted(n int) {
	d.mu.RLock()
	p := d.pipeline
	d.mu.RUnlock()

	if p != nil {
		p.SetLevelsWanted(n > 0)
	}
}

// streamStatus holds the connection open, writing one JSON event per line
// until the client goes away or the daemon stops.
func (d *Daemon) streamStatus(c net.Conn) {
	sub := d.hub.add()
	defer d.hub.remove(sub)

	enc := json.NewEncoder(c)

	// Send the current state straight away so a client that connects between
	// transitions renders correctly instead of sitting blank until the next
	// change.
	if err := enc.Encode(pipeline.StatusEvent{
		Status:    d.status(),
		Listening: d.listening(),
		At:        time.Now(),
	}); err != nil {
		return
	}

	// A client that has gone away is only discovered by writing to it, and a
	// silent pipeline produces nothing to write. This keeps a periodic
	// heartbeat flowing so those connections are reaped instead of
	// accumulating for the life of the daemon.
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case ev := <-sub.ch:
			if err := enc.Encode(ev); err != nil {
				return
			}

		case <-ticker.C:
			if err := enc.Encode(pipeline.StatusEvent{
				Status:    d.status(),
				Listening: d.listening(),
				At:        time.Now(),
			}); err != nil {
				return
			}

		case <-d.ctx.Done():
			return
		}
	}
}

// listening reports whether the microphone is currently open.
func (d *Daemon) listening() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.pipeline != nil && d.pipeline.Listening()
}
