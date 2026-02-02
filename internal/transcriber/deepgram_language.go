package transcriber

import "strings"

func normalizeDeepgramLanguage(code string) string {
	if code == "" {
		return ""
	}

	if strings.EqualFold(code, "en") || strings.EqualFold(code, "en-us") || strings.EqualFold(code, "en_us") {
		return "en-US"
	}

	return code
}
