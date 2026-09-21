package llm

import (
	"fmt"
	"strings"
)

// PostProcessingOptions controls which cleanup operations to request
type PostProcessingOptions struct {
	RemoveStutters    bool
	AddPunctuation    bool
	FixGrammar        bool
	RemoveFillerWords bool
}

// BuildSystemPrompt generates the system prompt for text cleanup.
// If override is non-empty, it replaces the built-in prompt body; the keywords
// line is still appended so domain terms reach the model regardless of which
// prompt is in use.
func BuildSystemPrompt(opts PostProcessingOptions, keywords []string, override string) string {
	var prompt string
	if override != "" {
		prompt = override
		// Ensure the keywords line below starts on its own line even if the
		// user's prompt didn't end with a newline.
		if !strings.HasSuffix(prompt, "\n") {
			prompt += "\n"
		}
	} else {
		prompt = buildDefaultSystemPrompt(opts)
	}

	if len(keywords) > 0 {
		prompt += fmt.Sprintf("\nContext keywords (use correct spelling for these terms): %s\n", strings.Join(keywords, ", "))
	}

	return prompt
}

func buildDefaultSystemPrompt(opts PostProcessingOptions) string {
	var tasks []string

	if opts.RemoveStutters {
		tasks = append(tasks, "Remove stutters and repeated words/phrases")
	}
	if opts.AddPunctuation {
		tasks = append(tasks, "Add proper punctuation")
	}
	if opts.FixGrammar {
		tasks = append(tasks, "Fix grammar errors")
	}
	if opts.RemoveFillerWords {
		tasks = append(tasks, "Remove filler words (um, uh, like, you know, etc.)")
	}

	if len(tasks) == 0 {
		tasks = append(tasks, "Clean up the text while preserving meaning")
	}

	prompt := "You are a text cleanup assistant. Your job is to clean up speech-to-text transcriptions.\n\n"
	prompt += "Tasks:\n"
	for _, task := range tasks {
		prompt += fmt.Sprintf("- %s\n", task)
	}

	prompt += "\nRules:\n"
	prompt += "- Preserve the original meaning and intent\n"
	prompt += "- Keep the same language as the input\n"
	prompt += "- Do not add any new information\n"
	prompt += "- Do not remove meaningful content\n"
	prompt += "- Output ONLY the cleaned text, nothing else\n"
	prompt += "- If the input is empty or nonsensical, return it as-is\n"

	return prompt
}

// DefaultSystemPrompt returns the built-in system prompt for the given options
// without appending the keywords line. Used to seed the TUI editor when the
// user opts in to a custom system prompt.
func DefaultSystemPrompt(opts PostProcessingOptions) string {
	return buildDefaultSystemPrompt(opts)
}

// renderSystemPrompt produces the final system prompt for this Config. It is
// pure with respect to Config, so adapters can render once at construction
// time rather than rebuilding the prompt on every Process call.
func (c Config) renderSystemPrompt() string {
	opts := PostProcessingOptions{
		RemoveStutters:    c.RemoveStutters,
		AddPunctuation:    c.AddPunctuation,
		FixGrammar:        c.FixGrammar,
		RemoveFillerWords: c.RemoveFillerWords,
	}
	return BuildSystemPrompt(opts, c.Keywords, c.SystemPrompt)
}

// BuildUserPrompt generates the user prompt with the text to process
func BuildUserPrompt(text string, customPrompt string) string {
	if customPrompt != "" {
		return fmt.Sprintf("%s\n\nText to process:\n%s", customPrompt, text)
	}
	return text
}
