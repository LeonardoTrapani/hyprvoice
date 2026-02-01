package tui

import (
	"strings"
	"testing"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

func TestGetTranscriptionModelOptions_ShowsCapabilities(t *testing.T) {
	// test elevenlabs - has batch-only and streaming-only models
	options := getTranscriptionModelOptions("elevenlabs", "")

	// should have 3 models: scribe_v1, scribe_v2, scribe_v2_realtime
	if len(options) != 3 {
		t.Errorf("expected 3 options for elevenlabs, got %d", len(options))
	}

	// verify models show capability tags
	for _, opt := range options {
		model, _, _ := provider.FindModelByID(opt.Value)
		if model == nil {
			continue
		}

		if model.SupportsStreaming && !model.SupportsBatch {
			// streaming-only should have [streaming] tag
			if !strings.Contains(opt.Key, "[streaming]") {
				t.Errorf("streaming-only model %s should have [streaming] tag in label: %s", opt.Value, opt.Key)
			}
		} else if model.SupportsBothModes() {
			// both modes should have [batch+streaming] tag
			if !strings.Contains(opt.Key, "[batch+streaming]") {
				t.Errorf("both-modes model %s should have [batch+streaming] tag in label: %s", opt.Value, opt.Key)
			}
		}
		// batch-only models don't need a tag
	}
}

func TestGetTranscriptionModelOptions_NoHeadersAnymore(t *testing.T) {
	// we removed batch/streaming section headers
	options := getTranscriptionModelOptions("elevenlabs", "")

	for _, opt := range options {
		if opt.Value == "" {
			t.Errorf("should not have headers anymore, got: %s", opt.Key)
		}
	}
}

func TestGetTranscriptionModelOptions_OpenAI_ShowsCapabilities(t *testing.T) {
	options := getTranscriptionModelOptions("openai", "")

	// OpenAI has 3 transcription models: whisper-1, gpt-4o-transcribe, gpt-4o-mini-transcribe
	if len(options) != 3 {
		t.Errorf("expected 3 options for openai, got %d", len(options))
	}

	// gpt-4o-transcribe and gpt-4o-mini-transcribe should have [batch+streaming]
	for _, opt := range options {
		if strings.Contains(opt.Value, "gpt-4o") {
			if !strings.Contains(opt.Key, "[batch+streaming]") {
				t.Errorf("gpt-4o model %s should have [batch+streaming] tag: %s", opt.Value, opt.Key)
			}
		}
	}
}

func TestGetTranscriptionModelOptions_Deepgram_ShowsBothModes(t *testing.T) {
	options := getTranscriptionModelOptions("deepgram", "")

	// Deepgram has 2 models: nova-3, nova-2 - both support batch+streaming
	if len(options) != 2 {
		t.Errorf("expected 2 options for deepgram, got %d", len(options))
	}

	for _, opt := range options {
		if !strings.Contains(opt.Key, "[batch+streaming]") {
			t.Errorf("deepgram model %s should have [batch+streaming] tag: %s", opt.Value, opt.Key)
		}
	}
}

func TestGetTranscriptionModelOptions_Groq_BatchOnly(t *testing.T) {
	// test groq - batch only (no streaming models)
	options := getTranscriptionModelOptions("groq-transcription", "")

	// should have 2 models: whisper-large-v3, whisper-large-v3-turbo
	if len(options) != 2 {
		t.Errorf("expected 2 options for groq, got %d", len(options))
	}

	// batch-only models should not have any mode tags
	for _, opt := range options {
		if strings.Contains(opt.Key, "[streaming]") || strings.Contains(opt.Key, "[batch]") {
			t.Errorf("batch-only model should not have mode tags: %s", opt.Key)
		}
	}
}
