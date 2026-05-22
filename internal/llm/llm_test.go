package llm

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt(t *testing.T) {
	tests := []struct {
		name        string
		opts        PostProcessingOptions
		keywords    []string
		override    string
		contains    []string
		notContains []string
	}{
		{
			name: "all options enabled",
			opts: PostProcessingOptions{
				RemoveStutters:    true,
				AddPunctuation:    true,
				FixGrammar:        true,
				RemoveFillerWords: true,
			},
			keywords: nil,
			contains: []string{
				"Remove stutters",
				"Add proper punctuation",
				"Fix grammar",
				"Remove filler words",
			},
		},
		{
			name: "only grammar",
			opts: PostProcessingOptions{
				FixGrammar: true,
			},
			keywords: nil,
			contains: []string{
				"Fix grammar",
			},
		},
		{
			name: "with keywords",
			opts: PostProcessingOptions{
				RemoveStutters: true,
			},
			keywords: []string{"Kubernetes", "TypeScript", "hyprvoice"},
			contains: []string{
				"Kubernetes",
				"TypeScript",
				"hyprvoice",
				"Context keywords",
			},
		},
		{
			name:     "no options - should have default",
			opts:     PostProcessingOptions{},
			keywords: nil,
			contains: []string{
				"Clean up the text",
			},
		},
		{
			name:     "override replaces built-in body",
			opts:     PostProcessingOptions{FixGrammar: true},
			keywords: nil,
			override: "You are a haiku poet. Reformat speech as haiku.",
			contains: []string{
				"haiku poet",
			},
			notContains: []string{
				"text cleanup assistant",
				"Fix grammar",
			},
		},
		{
			name:     "override still appends keywords",
			opts:     PostProcessingOptions{},
			keywords: []string{"Kubernetes"},
			override: "Custom system instructions.",
			contains: []string{
				"Custom system instructions.",
				"Context keywords",
				"Kubernetes",
			},
			notContains: []string{
				"text cleanup assistant",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildSystemPrompt(tc.opts, tc.keywords, tc.override)
			for _, expected := range tc.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected prompt to contain %q, got: %s", expected, result)
				}
			}
			for _, forbidden := range tc.notContains {
				if strings.Contains(result, forbidden) {
					t.Errorf("expected prompt NOT to contain %q, got: %s", forbidden, result)
				}
			}
		})
	}
}

func TestDefaultSystemPrompt(t *testing.T) {
	opts := PostProcessingOptions{FixGrammar: true}
	result := DefaultSystemPrompt(opts)
	if !strings.Contains(result, "Fix grammar") {
		t.Errorf("expected default to contain 'Fix grammar', got: %s", result)
	}
	if strings.Contains(result, "Context keywords") {
		t.Error("DefaultSystemPrompt should not append the keywords line")
	}
}

func TestAdaptersCacheSystemPromptAtConstruction(t *testing.T) {
	// The system prompt is a pure function of Config and should be rendered
	// once in NewXxxAdapter, not on every Process call. This test pins that
	// behavior so a regression that pushes work into the hot path is caught.
	cfg := Config{
		Provider:     "openai",
		APIKey:       "sk-test",
		Model:        "gpt-4o-mini",
		FixGrammar:   true,
		SystemPrompt: "you are a custom assistant",
	}

	openaiAdapter := NewOpenAIAdapter(cfg)
	if openaiAdapter.systemPrompt == "" {
		t.Error("OpenAIAdapter.systemPrompt should be populated at construction")
	}
	if !strings.Contains(openaiAdapter.systemPrompt, "custom assistant") {
		t.Errorf("OpenAIAdapter.systemPrompt should reflect override, got: %s", openaiAdapter.systemPrompt)
	}

	groqAdapter := NewGroqAdapter(Config{Provider: "groq", APIKey: "gsk-test", Model: "llama-3.3-70b-versatile", FixGrammar: true})
	if groqAdapter.systemPrompt == "" {
		t.Error("GroqAdapter.systemPrompt should be populated at construction")
	}
	if !strings.Contains(groqAdapter.systemPrompt, "Fix grammar") {
		t.Errorf("GroqAdapter.systemPrompt should include task line, got: %s", groqAdapter.systemPrompt)
	}
}

func TestBuildUserPrompt(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		customPrompt string
		expected     string
	}{
		{
			name:         "no custom prompt",
			text:         "hello world",
			customPrompt: "",
			expected:     "hello world",
		},
		{
			name:         "with custom prompt",
			text:         "hello world",
			customPrompt: "Format as a haiku",
			expected:     "Format as a haiku\n\nText to process:\nhello world",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildUserPrompt(tc.text, tc.customPrompt)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestNewAdapter(t *testing.T) {
	// Test OpenAI adapter creation
	openaiCfg := Config{
		Provider: "openai",
		APIKey:   "sk-test-key",
		Model:    "gpt-4o-mini",
	}
	adapter, err := NewAdapter(openaiCfg)
	if err != nil {
		t.Fatalf("failed to create openai adapter: %v", err)
	}
	if _, ok := adapter.(*OpenAIAdapter); !ok {
		t.Error("expected OpenAIAdapter type")
	}

	// Test Groq adapter creation
	groqCfg := Config{
		Provider: "groq",
		APIKey:   "gsk_test-key",
		Model:    "llama-3.3-70b-versatile",
	}
	adapter, err = NewAdapter(groqCfg)
	if err != nil {
		t.Fatalf("failed to create groq adapter: %v", err)
	}
	if _, ok := adapter.(*GroqAdapter); !ok {
		t.Error("expected GroqAdapter type")
	}

	// Test missing API key
	noKeyCfg := Config{
		Provider: "openai",
		APIKey:   "",
	}
	_, err = NewAdapter(noKeyCfg)
	if err == nil {
		t.Error("expected error for missing API key")
	}

	// Test unsupported provider
	badCfg := Config{
		Provider: "unsupported",
		APIKey:   "key",
	}
	_, err = NewAdapter(badCfg)
	if err == nil {
		t.Error("expected error for unsupported provider")
	}
}
