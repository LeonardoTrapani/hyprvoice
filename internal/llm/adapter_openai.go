package llm

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
)

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

	req := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.3, // Low temperature for consistent cleanup
	}

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
