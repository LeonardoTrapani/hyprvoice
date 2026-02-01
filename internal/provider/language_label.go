package provider

import (
	"fmt"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// LanguageLabel returns a human-readable label for a language code.
// Example: "es" -> "Spanish (es)", "en-US" -> "English (United States) (en-US)".
func LanguageLabel(code string) string {
	if code == "" {
		return ""
	}

	normalized := strings.ReplaceAll(code, "_", "-")
	tag, err := language.Parse(normalized)
	if err != nil {
		return fmt.Sprintf("language '%s'", code)
	}

	name := display.English.Tags().Name(tag)
	if name == "" || strings.EqualFold(name, code) {
		return fmt.Sprintf("language '%s'", code)
	}

	return fmt.Sprintf("%s (%s)", name, code)
}
