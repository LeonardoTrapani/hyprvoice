package transcriber

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// DeepgramBatchAdapter implements BatchAdapter for Deepgram pre-recorded transcription
type DeepgramBatchAdapter struct {
	endpoint *provider.EndpointConfig
	apiKey   string
	model    string
	language string
	keywords []string
}

// deepgramBatchResponse is the response from the pre-recorded API
type deepgramBatchResponse struct {
	Results *deepgramBatchResults `json:"results,omitempty"`
	Error   *deepgramError        `json:"error,omitempty"`
}

type deepgramBatchResults struct {
	Channels []deepgramBatchChannel `json:"channels,omitempty"`
}

type deepgramBatchChannel struct {
	Alternatives []deepgramAlternative `json:"alternatives,omitempty"`
}

// NewDeepgramBatchAdapter creates a new batch adapter for Deepgram
func NewDeepgramBatchAdapter(endpoint *provider.EndpointConfig, apiKey, model, lang string, keywords []string) *DeepgramBatchAdapter {
	return &DeepgramBatchAdapter{
		endpoint: endpoint,
		apiKey:   apiKey,
		model:    model,
		language: lang,
		keywords: keywords,
	}
}

// Transcribe sends audio data to Deepgram's pre-recorded API
func (a *DeepgramBatchAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if len(audioData) == 0 {
		return "", nil
	}

	// convert raw PCM to WAV format
	wavData, err := convertToWAV(audioData)
	if err != nil {
		return "", fmt.Errorf("convert to WAV: %w", err)
	}

	// build URL with query parameters
	apiURL, err := a.buildURL()
	if err != nil {
		return "", fmt.Errorf("build url: %w", err)
	}

	// create request with WAV data as body
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(wavData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// set headers
	req.Header.Set("Authorization", "Token "+a.apiKey)
	req.Header.Set("Content-Type", "audio/wav")

	// send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("deepgram api error (status %d): %s", resp.StatusCode, string(body))
	}

	// parse response
	var result deepgramBatchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("deepgram error: %s", result.Error.Message)
	}

	// extract transcript
	if result.Results == nil || len(result.Results.Channels) == 0 {
		return "", nil
	}
	if len(result.Results.Channels[0].Alternatives) == 0 {
		return "", nil
	}

	return result.Results.Channels[0].Alternatives[0].Transcript, nil
}

// buildURL constructs the API URL with query parameters
func (a *DeepgramBatchAdapter) buildURL() (string, error) {
	baseURL := a.endpoint.BaseURL + a.endpoint.Path

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base url: %w", err)
	}

	q := u.Query()
	q.Set("model", a.model)
	q.Set("smart_format", "true")
	q.Set("punctuate", "true")

	// add language if specified
	lang := normalizeDeepgramLanguage(a.language)
	if lang != "" {
		q.Set("language", lang)
	}

	// nova-3 uses "keyterm" (singular), others use "keywords" (plural)
	if len(a.keywords) > 0 && !strings.HasPrefix(a.model, "nova-3") && !strings.HasPrefix(a.model, "flux") {
		q.Set("keywords", strings.Join(a.keywords, ","))
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}
