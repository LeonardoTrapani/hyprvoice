package transcriber

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// WhisperServerAdapter implements BatchAdapter for the whisper.cpp HTTP server
// using its native /inference endpoint (multipart/form-data)
type WhisperServerAdapter struct {
	endpoint *provider.EndpointConfig
	model    string
	language string
}

func NewWhisperServerAdapter(endpoint *provider.EndpointConfig, model, language string) *WhisperServerAdapter {
	return &WhisperServerAdapter{
		endpoint: endpoint,
		model:    model,
		language: language,
	}
}

type whisperServerResponse struct {
	Text string `json:"text"`
}

var whisperServerHTTPClient = &http.Client{Timeout: 30 * time.Second}

func (a *WhisperServerAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if len(audioData) == 0 {
		return "", nil
	}

	wavData, err := convertToWAV(audioData)
	if err != nil {
		return "", fmt.Errorf("convert to WAV: %w", err)
	}

	// build multipart body
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	part, err := w.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(wavData); err != nil {
		return "", fmt.Errorf("write audio data: %w", err)
	}

	_ = w.WriteField("response_format", "json")
	_ = w.WriteField("temperature", "0.0")

	if a.model != "" {
		_ = w.WriteField("model", a.model)
	}
	if a.language != "" {
		_ = w.WriteField("language", a.language)
	}

	w.Close()

	url := strings.TrimRight(a.endpoint.BaseURL, "/") + a.endpoint.Path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	start := time.Now()
	resp, err := whisperServerHTTPClient.Do(req)
	duration := time.Since(start)
	if err != nil {
		return "", fmt.Errorf("whisper-server request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper-server error, status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	// whisper.cpp server may return one JSON object per segment on separate lines
	var parts []string
	dec := json.NewDecoder(bytes.NewReader(respBody))
	for {
		var result whisperServerResponse
		if err := dec.Decode(&result); err == io.EOF {
			break
		} else if err != nil {
			return "", fmt.Errorf("parse response: %w (body: %s)", err, string(respBody))
		}
		if t := strings.TrimSpace(result.Text); t != "" {
			parts = append(parts, t)
		}
	}

	text := strings.Join(parts, " ")
	log.Printf("whisper-server: transcribed %d bytes in %v: %q", len(audioData), duration, text)
	return text, nil
}
