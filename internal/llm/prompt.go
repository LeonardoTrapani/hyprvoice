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

// BuildSystemPrompt generates the system prompt for text cleanup
func BuildSystemPrompt(opts PostProcessingOptions, keywords []string) string {
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

	// If no tasks, just clean up generally
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

	if len(keywords) > 0 {
		prompt += fmt.Sprintf("\nContext keywords (use correct spelling for these terms): %s\n", strings.Join(keywords, ", "))
	}

	return prompt
}

// BuildUserPrompt generates the user prompt with the text to process
func BuildUserPrompt(text string, customPrompt string) string {
	if customPrompt != "" {
		return fmt.Sprintf("%s\n\nText to process:\n%s", customPrompt, text)
	}
	return text
}
