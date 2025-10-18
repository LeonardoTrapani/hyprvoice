package transcriber

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
)

// GroqTranscriptionAdapter implements TranscriptionAdapter for Groq Whisper API
type GroqTranscriptionAdapter struct {
	client *openai.Client
	config Config
}

func NewGroqTranscriptionAdapter(config Config) *GroqTranscriptionAdapter {
	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = "https://api.groq.com/openai/v1"
	client := openai.NewClientWithConfig(clientConfig)

	return &GroqTranscriptionAdapter{
		client: client,
		config: config,
	}
}

func (a *GroqTranscriptionAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if len(audioData) == 0 {
		return "", nil
	}

	// Convert raw PCM to WAV format
	wavData, err := convertToWAV(audioData)
	if err != nil {
		return "", fmt.Errorf("convert to WAV: %w", err)
	}

	// Create transcription request
	req := openai.AudioRequest{
		Model:    a.config.Model,
		Reader:   bytes.NewReader(wavData),
		FilePath: "audio.wav",
		Language: a.config.Language,
	}

	start := time.Now()
	resp, err := a.client.CreateTranscription(ctx, req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("groq-transcription-adapter: API call failed after %v: %v", duration, err)
		return "", fmt.Errorf("groq transcription: %w", err)
	}

	log.Printf("groq-transcription-adapter: transcribed %d bytes in %v: %q", len(audioData), duration, resp.Text)
	return resp.Text, nil
}

// convertToWAV converts raw 16-bit PCM audio to WAV format
func convertToWAV(rawAudio []byte) ([]byte, error) {
	var buf bytes.Buffer

	const sampleRate = 16000
	const channels = 1
	const bitsPerSample = 16
	const byteRate = sampleRate * channels * bitsPerSample / 8
	const blockAlign = channels * bitsPerSample / 8

	dataSize := len(rawAudio)
	fileSize := 36 + dataSize

	// WAV header
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(fileSize))
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))            // fmt chunk size
	binary.Write(&buf, binary.LittleEndian, uint16(1))             // PCM format
	binary.Write(&buf, binary.LittleEndian, uint16(channels))      // number of channels
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))    // sample rate
	binary.Write(&buf, binary.LittleEndian, uint32(byteRate))      // byte rate
	binary.Write(&buf, binary.LittleEndian, uint16(blockAlign))    // block align
	binary.Write(&buf, binary.LittleEndian, uint16(bitsPerSample)) // bits per sample

	// data chunk
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(dataSize))
	buf.Write(rawAudio)

	return buf.Bytes(), nil
}
