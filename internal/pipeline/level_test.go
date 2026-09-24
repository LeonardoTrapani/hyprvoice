package pipeline

import (
	"context"
	"encoding/binary"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/recording"
)

// pcm builds s16le bytes from the given samples.
func pcm(samples ...int16) []byte {
	out := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(s))
	}
	return out
}

// constantPCM builds n samples all at value.
func constantPCM(n int, value int16) []byte {
	samples := make([]int16, n)
	for i := range samples {
		samples[i] = value
	}
	return pcm(samples...)
}

func TestRMSSilenceIsZero(t *testing.T) {
	if got := rms(constantPCM(512, 0)); got != 0 {
		t.Fatalf("silence: got %v, want 0", got)
	}
}

func TestRMSFullScaleIsOne(t *testing.T) {
	got := rms(constantPCM(512, math.MaxInt16))
	if math.Abs(got-1) > 1e-9 {
		t.Fatalf("full scale: got %v, want 1", got)
	}
}

// A normalised meter must stay inside 0..1 even at the negative rail, which is
// one count louder than the positive one in two's complement.
func TestRMSNegativeRailStaysInRange(t *testing.T) {
	got := rms(constantPCM(512, math.MinInt16))
	if got < 0 || got > 1.0001 {
		t.Fatalf("negative rail: got %v, want within 0..1", got)
	}
}

func TestRMSHalfScale(t *testing.T) {
	got := rms(constantPCM(512, math.MaxInt16/2))
	if math.Abs(got-0.5) > 0.01 {
		t.Fatalf("half scale: got %v, want ~0.5", got)
	}
}

// RMS, not peak: a signal that is loud for half its samples and silent for the
// rest must read quieter than a constantly loud one.
func TestRMSTracksEnergyNotPeak(t *testing.T) {
	half := make([]int16, 512)
	for i := 0; i < 256; i++ {
		half[i] = math.MaxInt16
	}

	got := rms(pcm(half...))
	want := math.Sqrt(0.5)
	if math.Abs(got-want) > 0.01 {
		t.Fatalf("half-energy: got %v, want ~%v", got, want)
	}
}

func TestRMSEmptyWindow(t *testing.T) {
	if got := rms(nil); got != 0 {
		t.Fatalf("empty: got %v, want 0", got)
	}
}

func TestRMSLevelsSplitsFrameIntoWindows(t *testing.T) {
	const sampleRate, channels = 16000, 1

	// One default-sized frame: 8192 bytes = 4096 samples = 256ms.
	levels := rmsLevels(constantPCM(4096, 1000), sampleRate, channels)

	// 32ms windows at 16kHz mono = 512 samples each, so 256/32 = 8 of them.
	if len(levels) != 8 {
		t.Fatalf("got %d levels, want 8", len(levels))
	}
	for i, l := range levels {
		if l <= 0 || l > 1 {
			t.Fatalf("level %d out of range: %v", i, l)
		}
	}
}

func TestRMSLevelsHandlesPartialTrailingWindow(t *testing.T) {
	// 700 samples is one full 512-sample window plus a partial one.
	levels := rmsLevels(constantPCM(700, 5000), 16000, 1)
	if len(levels) != 2 {
		t.Fatalf("got %d levels, want 2", len(levels))
	}
}

func TestRMSLevelsStereoHalvesTheWindowCount(t *testing.T) {
	// Same byte count, two interleaved channels: half as much wall-clock
	// audio, so half as many windows.
	mono := rmsLevels(constantPCM(4096, 1000), 16000, 1)
	stereo := rmsLevels(constantPCM(4096, 1000), 16000, 2)

	if len(stereo) != len(mono)/2 {
		t.Fatalf("stereo produced %d levels, mono %d; want half", len(stereo), len(mono))
	}
}

func TestRMSLevelsRejectsBadConfig(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rate, ch int
	}{
		{"zero rate", 0, 1},
		{"zero channels", 16000, 0},
		{"negative rate", -1, 1},
	} {
		if got := rmsLevels(constantPCM(512, 1000), tc.rate, tc.ch); got != nil {
			t.Fatalf("%s: got %v, want nil", tc.name, got)
		}
	}
}

func TestRMSLevelsIgnoresStrayByte(t *testing.T) {
	if got := rmsLevels([]byte{0x01}, 16000, 1); got != nil {
		t.Fatalf("got %v, want nil for a sub-sample buffer", got)
	}
}

// A huge BufferSize must not produce an unbounded slice.
func TestRMSLevelsCapsPerFrame(t *testing.T) {
	levels := rmsLevels(constantPCM(512*(maxLevelsPerFrame+50), 1000), 16000, 1)
	if len(levels) > maxLevelsPerFrame {
		t.Fatalf("got %d levels, want at most %d", len(levels), maxLevelsPerFrame)
	}
}

// The tap sits in the audio path, so it must pass every frame through byte for
// byte. A dropped frame is a hole in the transcription.
func TestTapLevelsForwardsEveryFrame(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in := make(chan recording.AudioFrame, 4)
	out := tapLevels(ctx, in, 4, 16000, 1, func() bool { return true }, func([]float64) {})

	const frames = 16
	go func() {
		defer close(in)
		for i := 0; i < frames; i++ {
			in <- recording.AudioFrame{Data: constantPCM(512, int16(i+1)), Timestamp: time.Now()}
		}
	}()

	var got int
	for frame := range out {
		want := constantPCM(512, int16(got+1))
		if string(frame.Data) != string(want) {
			t.Fatalf("frame %d altered in transit", got)
		}
		got++
	}

	if got != frames {
		t.Fatalf("forwarded %d frames, want %d", got, frames)
	}
}

func TestTapLevelsSkipsWorkWhenUnwatched(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var emitted atomic.Int64
	in := make(chan recording.AudioFrame, 4)
	out := tapLevels(ctx, in, 4, 16000, 1,
		func() bool { return false },
		func([]float64) { emitted.Add(1) },
	)

	in <- recording.AudioFrame{Data: constantPCM(4096, 1000)}
	close(in)
	for range out { //nolint:revive // draining
	}

	if n := emitted.Load(); n != 0 {
		t.Fatalf("emitted %d level events with no subscriber, want 0", n)
	}
}

func TestTapLevelsEmitsWhenWatched(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	levels := make(chan []float64, 8)
	in := make(chan recording.AudioFrame, 4)
	out := tapLevels(ctx, in, 4, 16000, 1,
		func() bool { return true },
		func(l []float64) { levels <- l },
	)

	in <- recording.AudioFrame{Data: constantPCM(4096, 1000)}
	close(in)
	for range out { //nolint:revive // draining
	}

	// Batched, so the 8 windows may arrive across one or more events; the
	// flush deferred on shutdown guarantees none are lost.
	var got int
	for {
		select {
		case l := <-levels:
			got += len(l)
			continue
		default:
		}
		break
	}

	if got != 8 {
		t.Fatalf("got %d levels in total, want 8", got)
	}
}

// Frame size is set by PipeWire, not by us, so the publish rate must come from
// the clock rather than from how often frames happen to arrive.
func TestTapLevelsBatchesRapidFrames(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var events atomic.Int64
	in := make(chan recording.AudioFrame, 64)
	out := tapLevels(ctx, in, 64, 16000, 1,
		func() bool { return true },
		func([]float64) { events.Add(1) },
	)

	// 60 frames of one 32ms window each, delivered as fast as they can be
	// read. Per-frame emission would produce 60 events.
	const frames = 60
	go func() {
		defer close(in)
		for i := 0; i < frames; i++ {
			in <- recording.AudioFrame{Data: constantPCM(512, 8000)}
		}
	}()
	for range out { //nolint:revive // draining
	}

	if n := events.Load(); n >= frames {
		t.Fatalf("emitted %d events for %d frames; batching did not apply", n, frames)
	}
}

// Levels measured but not yet published must not be lost when recording ends.
func TestTapLevelsFlushesPendingOnClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	levels := make(chan []float64, 8)
	in := make(chan recording.AudioFrame, 4)
	out := tapLevels(ctx, in, 4, 16000, 1,
		func() bool { return true },
		func(l []float64) { levels <- l },
	)

	// A single short frame, well inside the flush interval, then close.
	in <- recording.AudioFrame{Data: constantPCM(512, 9000)}
	close(in)
	for range out { //nolint:revive // draining
	}

	select {
	case l := <-levels:
		if len(l) != 1 {
			t.Fatalf("got %d levels, want 1", len(l))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pending levels were dropped instead of flushed on close")
	}
}

func TestTapLevelsStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	in := make(chan recording.AudioFrame)
	out := tapLevels(ctx, in, 1, 16000, 1, func() bool { return true }, func([]float64) {})

	cancel()

	select {
	case _, ok := <-out:
		if ok {
			t.Fatal("received a frame after cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("tap did not shut down on context cancel")
	}
}

// The tap runs on every captured frame, so its cost needs to stay invisible
// next to the transcription it feeds.
func BenchmarkRMSLevelsDefaultFrame(b *testing.B) {
	frame := constantPCM(4096, 12000)
	b.SetBytes(int64(len(frame)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rmsLevels(frame, 16000, 1)
	}
}
