package llm

import (
	"context"
	"fmt"
)

// Adapter interface for LLM text processing
type Adapter interface {
	Process(ctx context.Context, text string) (string, error)
}

// Config holds LLM adapter configuration
type Config struct {
	Provider          string
	APIKey            string
	Model             string
	RemoveStutters    bool
	AddPunctuation    bool
	FixGrammar        bool
	RemoveFillerWords bool
	CustomPrompt      string
	Keywords          []string
}

// NewAdapter creates an LLM adapter based on the provider
func NewAdapter(cfg Config) (Adapter, error) {
	switch cfg.Provider {
	case "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key required")
		}
		return NewOpenAIAdapter(cfg), nil
	case "groq":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("Groq API key required")
		}
		return NewGroqAdapter(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}
