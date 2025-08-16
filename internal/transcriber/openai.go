package transcriber

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/recording"
	"github.com/sashabaranov/go-openai"
)

type OpenAITranscriber struct {
	client       *openai.Client
	config       Config
	buffer       *audioBuffer
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.Mutex
	transcribing bool

	transcriptionMu   sync.RWMutex
	transcriptionText strings.Builder
}

type audioBuffer struct {
	data    []byte
	mu      sync.Mutex
	lastAdd time.Time
	maxSize int
}

func NewOpenAITranscriber(config Config) *OpenAITranscriber {
	client := openai.NewClient(config.APIKey)

	buffer := &audioBuffer{
		data:    make([]byte, 0, config.ChunkSize*2),
		maxSize: config.ChunkSize,
	}

	return &OpenAITranscriber{
		client: client,
		config: config,
		buffer: buffer,
	}
}

func (t *OpenAITranscriber) Start(ctx context.Context, frameCh <-chan recording.AudioFrame) (<-chan error, error) {
	t.mu.Lock()
	if t.transcribing {
		t.mu.Unlock()
		return nil, fmt.Errorf("transcriber: already transcribing")
	}
	t.transcribing = true
	t.mu.Unlock()

	transcribeCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	errCh := make(chan error, 1)

	t.wg.Add(1)
	go t.processFrames(transcribeCtx, frameCh, errCh)

	return errCh, nil
}

func (t *OpenAITranscriber) Stop() error {
	t.mu.Lock()
	if !t.transcribing {
		t.mu.Unlock()
		return nil
	}
	cancel := t.cancel
	t.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	t.wg.Wait()
	return nil
}

func (t *OpenAITranscriber) GetTranscription() (string, error) {
	t.transcriptionMu.RLock()
	defer t.transcriptionMu.RUnlock()

	if t.transcriptionText.Len() == 0 {
		return "", nil
	}

	return t.transcriptionText.String(), nil
}

func (t *OpenAITranscriber) processFrames(ctx context.Context, frameCh <-chan recording.AudioFrame, errCh chan<- error) {
	defer func() {
		close(errCh)
		t.mu.Lock()
		t.transcribing = false
		t.cancel = nil
		t.mu.Unlock()
		t.wg.Done()
	}()

	ticker := time.NewTicker(t.config.BufferTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if t.buffer.hasData() {
				t.transcribeBuffer(ctx, errCh)
			}
			return

		case frame, ok := <-frameCh:
			if !ok {
				if t.buffer.hasData() {
					t.transcribeBuffer(ctx, errCh)
				}
				return
			}
			t.buffer.addFrame(frame)

		case <-ticker.C:
			if t.buffer.shouldFlush(t.config.BufferTime) {
				t.transcribeBuffer(ctx, errCh)
			}
		}
	}
}

func (t *OpenAITranscriber) transcribeBuffer(ctx context.Context, errCh chan<- error) {
	audioData := t.buffer.flush()
	if len(audioData) == 0 {
		return
	}

	log.Printf("transcriber: sending %d bytes to OpenAI API", len(audioData))

	wavData, err := t.convertToWAV(audioData)
	if err != nil {
		log.Printf("transcriber: failed to convert audio to WAV: %v", err)
		select {
		case errCh <- fmt.Errorf("transcriber: convert to WAV: %w", err):
		default:
		}
		return
	}

	req := openai.AudioRequest{
		Model:    t.config.Model,
		Reader:   bytes.NewReader(wavData),
		FilePath: "audio.wav",
		Language: t.config.Language,
	}

	start := time.Now()
	resp, err := t.client.CreateTranscription(ctx, req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("transcriber: API call failed after %v: %v", duration, err)
		select {
		case errCh <- fmt.Errorf("transcriber: transcription failed: %w", err):
		default:
		}
		return
	}

	if resp.Text != "" {
		log.Printf("transcriber: received result in %v: %q", duration, resp.Text)
		t.transcriptionMu.Lock()
		if t.transcriptionText.Len() > 0 {
			t.transcriptionText.WriteString(" ")
		}
		t.transcriptionText.WriteString(strings.TrimSpace(resp.Text))
		t.transcriptionMu.Unlock()
	} else {
		log.Printf("transcriber: received empty result after %v", duration)
	}
}

func (t *OpenAITranscriber) convertToWAV(rawAudio []byte) ([]byte, error) {
	var buf bytes.Buffer

	const sampleRate = 16000
	const channels = 1
	const bitsPerSample = 16
	const byteRate = sampleRate * channels * bitsPerSample / 8
	const blockAlign = channels * bitsPerSample / 8

	dataSize := len(rawAudio)
	fileSize := 36 + dataSize

	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(fileSize))
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(channels))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(&buf, binary.LittleEndian, uint32(byteRate))
	binary.Write(&buf, binary.LittleEndian, uint16(blockAlign))
	binary.Write(&buf, binary.LittleEndian, uint16(bitsPerSample))

	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(dataSize))
	buf.Write(rawAudio)

	return buf.Bytes(), nil
}

func (b *audioBuffer) addFrame(frame recording.AudioFrame) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.data = append(b.data, frame.Data...)
	b.lastAdd = frame.Timestamp

	if len(b.data) > b.maxSize*2 {
		b.data = b.data[len(b.data)-b.maxSize:]
	}
}

func (b *audioBuffer) flush() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.data) == 0 {
		return nil
	}

	result := make([]byte, len(b.data))
	copy(result, b.data)
	b.data = b.data[:0]

	return result
}

func (b *audioBuffer) hasData() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.data) > 0
}

func (b *audioBuffer) shouldFlush(bufferTime time.Duration) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.data) == 0 {
		return false
	}

	if len(b.data) >= b.maxSize {
		return true
	}

	return time.Since(b.lastAdd) >= bufferTime
}

func NewTranscriber(config Config) (Transcriber, error) {
	switch config.Provider {
	case "openai":
		if config.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key required")
		}
		return NewOpenAITranscriber(config), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}
