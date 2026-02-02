package transcriber

import "errors"

// FatalTranscriptionError marks an error as non-recoverable for the current session.
type FatalTranscriptionError struct {
	Err error
}

func (e *FatalTranscriptionError) Error() string {
	if e == nil || e.Err == nil {
		return "fatal transcription error"
	}
	return e.Err.Error()
}

func (e *FatalTranscriptionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewFatalTranscriptionError(err error) error {
	if err == nil {
		return nil
	}
	return &FatalTranscriptionError{Err: err}
}

func IsFatalTranscriptionError(err error) bool {
	var fatal *FatalTranscriptionError
	return errors.As(err, &fatal)
}
