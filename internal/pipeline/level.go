package pipeline

import (
	"context"
	"encoding/binary"
	"math"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/recording"
)

// levelWindowMs is the slice of audio each emitted level summarises. The
// recorder hands us one frame per BufferSize bytes, which at the default 8 KiB
// and 16 kHz mono is 256 ms -- far too coarse for a meter to look alive. Each
// frame is therefore subdivided into windows of this length, so a single frame
// yields a short run of levels the UI can scroll through smoothly. The cost is
// one pass over bytes we have already copied.
const levelWindowMs = 32

// maxLevelsPerFrame caps the run emitted for one frame so a pathologically
// large BufferSize cannot produce an unbounded slice.
const maxLevelsPerFrame = 64

// levelFlushInterval bounds how often level events are published.
//
// Frame size is not ours to choose: the recorder reads whatever PipeWire has
// ready rather than filling BufferSize, which in practice means a frame every
// ~32ms. Emitting per frame would tie the event rate to that chunking and make
// it drift with the audio config. Accumulating instead keeps publication at a
// predictable ~20/s while still carrying every window, so the UI loses no
// resolution.
const levelFlushInterval = 50 * time.Millisecond

// rmsLevels summarises s16 PCM as a series of normalised 0..1 loudness values,
// one per levelWindowMs of audio.
//
// Root mean square rather than peak: peak tracks single samples and jitters
// wildly between windows, while RMS follows perceived loudness and produces a
// meter that reads as the voice rather than as noise.
func rmsLevels(data []byte, sampleRate, channels int) []float64 {
	if sampleRate <= 0 || channels <= 0 || len(data) < 2 {
		return nil
	}

	// s16: two bytes per sample, interleaved per channel.
	samplesPerWindow := sampleRate * channels * levelWindowMs / 1000
	if samplesPerWindow <= 0 {
		return nil
	}
	bytesPerWindow := samplesPerWindow * 2

	levels := make([]float64, 0, len(data)/bytesPerWindow+1)
	for start := 0; start < len(data) && len(levels) < maxLevelsPerFrame; start += bytesPerWindow {
		end := min(start+bytesPerWindow, len(data))

		// A trailing partial window is still worth reporting, but a single
		// stray byte carries no sample.
		if end-start < 2 {
			break
		}
		levels = append(levels, rms(data[start:end]))
	}
	return levels
}

// rms returns the root mean square of one window of s16 PCM, normalised to
// 0..1 against full scale.
func rms(window []byte) float64 {
	n := len(window) / 2
	if n == 0 {
		return 0
	}

	var sum float64
	for i := 0; i < n; i++ {
		// The recorder requests s16 little-endian from PipeWire.
		s := float64(int16(binary.LittleEndian.Uint16(window[i*2:])))
		sum += s * s
	}

	// math.MaxInt16 rather than 32768: a normalised value of exactly 1.0 for a
	// full-scale signal keeps the UI's 0..1 contract honest.
	return math.Sqrt(sum/float64(n)) / float64(math.MaxInt16)
}

// tapLevels forwards frames from in to the returned channel, emitting a run of
// loudness levels for each one via emit.
//
// Forwarding blocks rather than dropping. The recorder already sheds frames
// when its own channel backs up, so blocking here simply lets that existing
// backpressure reach it; dropping instead would punch holes in the audio the
// transcriber receives.
func tapLevels(
	ctx context.Context,
	in <-chan recording.AudioFrame,
	capacity, sampleRate, channels int,
	wanted func() bool,
	emit func([]float64),
) <-chan recording.AudioFrame {
	out := make(chan recording.AudioFrame, capacity)

	go func() {
		defer close(out)

		var pending []float64
		lastFlush := time.Now()

		flush := func() {
			if len(pending) == 0 {
				return
			}
			emit(pending)
			pending = nil
			lastFlush = time.Now()
		}

		// Anything measured since the last flush still belongs to the user's
		// speech, so publish it rather than discarding it on the way out.
		defer flush()

		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-in:
				if !ok {
					return
				}

				// No subscriber: skip the arithmetic entirely, leaving the tap
				// as a single channel hop per frame.
				if wanted() {
					pending = append(pending, rmsLevels(frame.Data, sampleRate, channels)...)
					if time.Since(lastFlush) >= levelFlushInterval {
						flush()
					}
				} else if pending != nil {
					// Stopped being watched mid-recording.
					pending = nil
				}

				select {
				case out <- frame:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out
}
