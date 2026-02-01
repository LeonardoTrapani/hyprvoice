package tui

import (
	"testing"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

func TestGetTranscriptionModelOptions_GroupsModels(t *testing.T) {
	// test elevenlabs - has both batch and streaming
	options := getTranscriptionModelOptions("elevenlabs", "")

	// find headers
	var batchHeaderIdx, streamingHeaderIdx int
	batchHeaderIdx = -1
	streamingHeaderIdx = -1

	for i, opt := range options {
		if opt.Value == "" {
			if opt.Key == "─── Batch ───" {
				batchHeaderIdx = i
			}
			if opt.Key == "─── Streaming ───" {
				streamingHeaderIdx = i
			}
		}
	}

	if batchHeaderIdx == -1 {
		t.Error("expected Batch header for provider with both types")
	}
	if streamingHeaderIdx == -1 {
		t.Error("expected Streaming header for provider with both types")
	}
	if batchHeaderIdx >= streamingHeaderIdx {
		t.Errorf("Batch header should come before Streaming header: batch=%d, streaming=%d", batchHeaderIdx, streamingHeaderIdx)
	}

	// verify models are grouped correctly
	for i, opt := range options {
		if opt.Value == "" {
			continue // skip headers
		}
		model, _, _ := provider.FindModelByID(opt.Value)
		if model == nil {
			continue // unknown model
		}

		if i < streamingHeaderIdx && model.Streaming {
			t.Errorf("streaming model %s found before streaming header", opt.Value)
		}
		if i > streamingHeaderIdx && !model.Streaming {
			t.Errorf("batch model %s found after streaming header", opt.Value)
		}
	}
}

func TestGetTranscriptionModelOptions_NoHeadersForSingleType(t *testing.T) {
	// test groq - batch only (no streaming models)
	options := getTranscriptionModelOptions("groq-transcription", "")

	for _, opt := range options {
		if opt.Value == "" {
			t.Errorf("expected no headers for provider with only one model type, got: %s", opt.Key)
		}
	}
}

func TestGetTranscriptionModelOptions_OpenAI_GroupsCorrectly(t *testing.T) {
	options := getTranscriptionModelOptions("openai", "")

	var batchHeaderIdx, streamingHeaderIdx int
	batchHeaderIdx = -1
	streamingHeaderIdx = -1

	for i, opt := range options {
		if opt.Value == "" {
			if opt.Key == "─── Batch ───" {
				batchHeaderIdx = i
			}
			if opt.Key == "─── Streaming ───" {
				streamingHeaderIdx = i
			}
		}
	}

	// OpenAI has 3 batch + 1 streaming
	if batchHeaderIdx == -1 {
		t.Error("expected Batch header for OpenAI")
	}
	if streamingHeaderIdx == -1 {
		t.Error("expected Streaming header for OpenAI")
	}

	// count models (not headers) by position
	batchCount := 0
	streamingCount := 0
	for i, opt := range options {
		if opt.Value == "" {
			continue // skip headers
		}
		if i > batchHeaderIdx && i < streamingHeaderIdx {
			batchCount++
		} else if i > streamingHeaderIdx {
			streamingCount++
		}
	}

	if batchCount < 3 {
		t.Errorf("expected at least 3 batch models for OpenAI, got %d", batchCount)
	}
	if streamingCount < 1 {
		t.Errorf("expected at least 1 streaming model for OpenAI, got %d", streamingCount)
	}
}
