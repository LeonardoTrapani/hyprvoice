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
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// ElevenLabsAdapter implements BatchAdapter for ElevenLabs Scribe API
type ElevenLabsAdapter struct {
	client   *http.Client
	endpoint *provider.EndpointConfig
	apiKey   string
	model    string
	language string
	keywords []string
}

// ElevenLabsResponse represents the API response
type ElevenLabsResponse struct {
	Text string `json:"text"`
}

// NewElevenLabsAdapter creates an adapter for ElevenLabs Scribe API
// endpoint: the endpoint config (BaseURL + Path)
// apiKey: ElevenLabs API key
// model: model ID (e.g., "scribe_v1")
// lang: provider language code
func NewElevenLabsAdapter(endpoint *provider.EndpointConfig, apiKey, model, lang string, keywords []string) *ElevenLabsAdapter {
	return &ElevenLabsAdapter{
		client:   &http.Client{Timeout: 30 * time.Second},
		endpoint: endpoint,
		apiKey:   apiKey,
		model:    model,
		language: lang,
		keywords: keywords,
	}
}

// Transcribe sends audio to ElevenLabs API for transcription
func (a *ElevenLabsAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if len(audioData) == 0 {
		return "", nil
	}

	// Convert raw PCM to WAV format
	wavData, err := convertToWAV(audioData)
	if err != nil {
		return "", fmt.Errorf("convert to WAV: %w", err)
	}

	// Create multipart form body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add audio file
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(wavData)); err != nil {
		return "", fmt.Errorf("copy audio data: %w", err)
	}

	// Add model_id
	if err := writer.WriteField("model_id", a.model); err != nil {
		return "", fmt.Errorf("write model_id: %w", err)
	}

	// Add language_code if specified
	if a.language != "" {
		if err := writer.WriteField("language_code", a.language); err != nil {
			return "", fmt.Errorf("write language_code: %w", err)
		}
	}

	if len(a.keywords) > 0 {
		keytermsJSON, err := json.Marshal(a.keywords)
		if err != nil {
			return "", fmt.Errorf("marshal keyterms: %w", err)
		}
		if err := writer.WriteField("keyterms", string(keytermsJSON)); err != nil {
			return "", fmt.Errorf("write keyterms: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close writer: %w", err)
	}

	// Create HTTP request using endpoint config
	url := a.endpoint.BaseURL + a.endpoint.Path
	req, err := http.NewRequestWithContext(ctx, "POST", url, &body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("xi-api-key", a.apiKey)

	start := time.Now()
	resp, err := a.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("elevenlabs-adapter: API call failed after %v: %v", duration, err)
		return "", fmt.Errorf("elevenlabs request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("elevenlabs-adapter: API returned status %d: %s", resp.StatusCode, string(bodyBytes))
		return "", fmt.Errorf("elevenlabs API status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result ElevenLabsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	log.Printf("elevenlabs-adapter: transcribed %d bytes in %v: %q", len(audioData), duration, result.Text)
	return result.Text, nil
}
