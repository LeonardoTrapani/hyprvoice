package llm

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
)

// GroqAdapter implements Adapter using Groq's OpenAI-compatible API
type GroqAdapter struct {
	client *openai.Client
	config Config
}

// NewGroqAdapter creates a new Groq LLM adapter
func NewGroqAdapter(cfg Config) *GroqAdapter {
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	clientConfig.BaseURL = "https://api.groq.com/openai/v1"
	return &GroqAdapter{
		client: openai.NewClientWithConfig(clientConfig),
		config: cfg,
	}
}

func (a *GroqAdapter) Process(ctx context.Context, text string) (string, error) {
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
		model = "llama-3.3-70b-versatile"
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
		log.Printf("groq-llm-adapter: API call failed after %v: %v", duration, err)
		return "", fmt.Errorf("groq chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("groq chat completion: no response choices")
	}

	result := resp.Choices[0].Message.Content
	log.Printf("groq-llm-adapter: processed in %v: %q -> %q", duration, text, result)
	return result, nil
}
