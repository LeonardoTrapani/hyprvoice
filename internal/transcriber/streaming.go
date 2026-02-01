package transcriber

import "context"

// TranscriptionResult represents a single transcription result from a streaming adapter
type TranscriptionResult struct {
	Text    string // the transcription text (partial or final)
	IsFinal bool   // true if this is a final result, false for interim results
	Error   error  // non-nil if an error occurred
}

// StreamingAdapter interface for streaming transcription backends (send audio in real-time)
type StreamingAdapter interface {
	// Start initiates the streaming connection with the given language setting
	Start(ctx context.Context, language string) error

	// SendChunk sends a chunk of audio data to the transcription service
	SendChunk(audio []byte) error

	// Results returns a channel that receives transcription results (partial and final)
	Results() <-chan TranscriptionResult

	// Finalize signals end of audio input and waits for final transcription results.
	// This should be called before Close to ensure all pending audio is committed.
	// The ctx controls the timeout for waiting on final results.
	Finalize(ctx context.Context) error

	// Close gracefully closes the streaming connection
	Close() error
}
