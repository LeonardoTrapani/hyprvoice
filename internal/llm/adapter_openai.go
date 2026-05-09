package llm

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// buildOpenAIChatRequest builds the chat completion request for an OpenAI
// model. If the registry marks the model with RestrictedSampling, Temperature
// is left unset so the API doesn't reject the request with the
// "temperature ... fixed at 1" error. Models not in the registry fall through
// to the default Temperature=0.3, which keeps existing behavior for any future
// gpt-4o variant a user might try before it's added here.
func buildOpenAIChatRequest(model, systemPrompt, userPrompt string) openai.ChatCompletionRequest {
	req := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
	}
	if m, err := provider.GetModel(provider.ProviderOpenAI, model); err == nil && m.RestrictedSampling {
		return req
	}
	req.Temperature = 0.3 // Low temperature for consistent cleanup
	return req
}

// OpenAIAdapter implements Adapter using OpenAI's chat completions API
type OpenAIAdapter struct {
	client *openai.Client
	config Config
}

// NewOpenAIAdapter creates a new OpenAI LLM adapter
func NewOpenAIAdapter(cfg Config) *OpenAIAdapter {
	return &OpenAIAdapter{
		client: openai.NewClient(cfg.APIKey),
		config: cfg,
	}
}

func (a *OpenAIAdapter) Process(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", nil
	}

	opts := PostProcessingOptions{
		RemoveStutters:    a.config.RemoveStutters,
		AddPunctuation:    a.config.AddPunctuation,
		FixGrammar:        a.config.FixGrammar,
		RemoveFillerWords: a.config.RemoveFillerWords,
	}

	systemPrompt := BuildSystemPrompt(opts, a.config.Keywords)
	userPrompt := BuildUserPrompt(text, a.config.CustomPrompt)

	model := a.config.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	req := buildOpenAIChatRequest(model, systemPrompt, userPrompt)

	start := time.Now()
	resp, err := a.client.CreateChatCompletion(ctx, req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("openai-llm-adapter: API call failed after %v: %v", duration, err)
		return "", fmt.Errorf("openai chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai chat completion: no response choices")
	}

	result := resp.Choices[0].Message.Content
	log.Printf("openai-llm-adapter: processed in %v: %q -> %q", duration, text, result)
	return result, nil
}
