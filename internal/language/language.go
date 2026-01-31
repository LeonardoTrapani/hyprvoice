package language

// Language represents a supported transcription language
type Language struct {
	Code       string // ISO 639-1 code (e.g., "en", "es", "zh")
	Name       string // English name (e.g., "English", "Spanish")
	NativeName string // Native name (e.g., "English", "Espanol", "中文")
}

// Auto represents auto-detection - used when user doesn't specify a language
var Auto = Language{Code: "", Name: "Auto-detect", NativeName: ""}

// languages is the master list of supported languages
// derived from OpenAI Whisper's 57 supported languages
var languages = []Language{
	{Code: "af", Name: "Afrikaans", NativeName: "Afrikaans"},
	{Code: "ar", Name: "Arabic", NativeName: "العربية"},
	{Code: "hy", Name: "Armenian", NativeName: "Հdelays"},
	{Code: "az", Name: "Azerbaijani", NativeName: "Azərbaycan"},
	{Code: "be", Name: "Belarusian", NativeName: "Беларуская"},
	{Code: "bs", Name: "Bosnian", NativeName: "Bosanski"},
	{Code: "bg", Name: "Bulgarian", NativeName: "Български"},
	{Code: "ca", Name: "Catalan", NativeName: "Català"},
	{Code: "zh", Name: "Chinese", NativeName: "中文"},
	{Code: "hr", Name: "Croatian", NativeName: "Hrvatski"},
	{Code: "cs", Name: "Czech", NativeName: "Čeština"},
	{Code: "da", Name: "Danish", NativeName: "Dansk"},
	{Code: "nl", Name: "Dutch", NativeName: "Nederlands"},
	{Code: "en", Name: "English", NativeName: "English"},
	{Code: "et", Name: "Estonian", NativeName: "Eesti"},
	{Code: "fi", Name: "Finnish", NativeName: "Suomi"},
	{Code: "fr", Name: "French", NativeName: "Français"},
	{Code: "gl", Name: "Galician", NativeName: "Galego"},
	{Code: "de", Name: "German", NativeName: "Deutsch"},
	{Code: "el", Name: "Greek", NativeName: "Ελληνικά"},
	{Code: "he", Name: "Hebrew", NativeName: "עברית"},
	{Code: "hi", Name: "Hindi", NativeName: "हिन्दी"},
	{Code: "hu", Name: "Hungarian", NativeName: "Magyar"},
	{Code: "is", Name: "Icelandic", NativeName: "Íslenska"},
	{Code: "id", Name: "Indonesian", NativeName: "Bahasa Indonesia"},
	{Code: "it", Name: "Italian", NativeName: "Italiano"},
	{Code: "ja", Name: "Japanese", NativeName: "日本語"},
	{Code: "kn", Name: "Kannada", NativeName: "ಕನ್ನಡ"},
	{Code: "kk", Name: "Kazakh", NativeName: "Қазақ"},
	{Code: "ko", Name: "Korean", NativeName: "한국어"},
	{Code: "lv", Name: "Latvian", NativeName: "Latviešu"},
	{Code: "lt", Name: "Lithuanian", NativeName: "Lietuvių"},
	{Code: "mk", Name: "Macedonian", NativeName: "Македонски"},
	{Code: "ms", Name: "Malay", NativeName: "Bahasa Melayu"},
	{Code: "mr", Name: "Marathi", NativeName: "मराठी"},
	{Code: "mi", Name: "Maori", NativeName: "Māori"},
	{Code: "ne", Name: "Nepali", NativeName: "नेपाली"},
	{Code: "no", Name: "Norwegian", NativeName: "Norsk"},
	{Code: "fa", Name: "Persian", NativeName: "فارسی"},
	{Code: "pl", Name: "Polish", NativeName: "Polski"},
	{Code: "pt", Name: "Portuguese", NativeName: "Português"},
	{Code: "ro", Name: "Romanian", NativeName: "Română"},
	{Code: "ru", Name: "Russian", NativeName: "Русский"},
	{Code: "sr", Name: "Serbian", NativeName: "Српски"},
	{Code: "sk", Name: "Slovak", NativeName: "Slovenčina"},
	{Code: "sl", Name: "Slovenian", NativeName: "Slovenščina"},
	{Code: "es", Name: "Spanish", NativeName: "Español"},
	{Code: "sw", Name: "Swahili", NativeName: "Kiswahili"},
	{Code: "sv", Name: "Swedish", NativeName: "Svenska"},
	{Code: "tl", Name: "Tagalog", NativeName: "Tagalog"},
	{Code: "ta", Name: "Tamil", NativeName: "தமிழ்"},
	{Code: "th", Name: "Thai", NativeName: "ไทย"},
	{Code: "tr", Name: "Turkish", NativeName: "Türkçe"},
	{Code: "uk", Name: "Ukrainian", NativeName: "Українська"},
	{Code: "ur", Name: "Urdu", NativeName: "اردو"},
	{Code: "vi", Name: "Vietnamese", NativeName: "Tiếng Việt"},
	{Code: "cy", Name: "Welsh", NativeName: "Cymraeg"},
}

// codeIndex maps language codes to their Language structs for fast lookup
var codeIndex map[string]Language

func init() {
	codeIndex = make(map[string]Language, len(languages)+1)
	codeIndex[""] = Auto // auto-detect is valid
	for _, lang := range languages {
		codeIndex[lang.Code] = lang
	}
}

// FromCode returns the Language for the given code.
// Returns Auto if code is not found.
func FromCode(code string) Language {
	if lang, ok := codeIndex[code]; ok {
		return lang
	}
	return Auto
}

// List returns all supported languages (excluding Auto)
func List() []Language {
	result := make([]Language, len(languages))
	copy(result, languages)
	return result
}

// Codes returns all language codes (excluding empty string for auto)
func Codes() []string {
	codes := make([]string, len(languages))
	for i, lang := range languages {
		codes[i] = lang.Code
	}
	return codes
}

// AllLanguageCodes is an alias for Codes - used by models that support all languages
func AllLanguageCodes() []string {
	return Codes()
}

// IsValidCode returns true if the code is recognized (including empty for auto)
func IsValidCode(code string) bool {
	_, ok := codeIndex[code]
	return ok
}

// ToProviderFormat converts a canonical language code to the format expected by a specific provider.
// Each provider may have different expectations:
//   - whisper-cpp: uses standard codes like 'en', 'auto' for auto-detect
//   - openai: uses standard codes like 'en', empty string for auto-detect
//   - groq: same as openai (OpenAI-compatible)
//   - mistral: same as openai (OpenAI-compatible)
//   - deepgram: uses locale codes like 'en-US', 'es' for Spanish
//   - elevenlabs: uses standard codes or full names depending on API version
func ToProviderFormat(code string, providerName string) string {
	// handle auto-detect (empty code)
	if code == "" {
		switch providerName {
		case "whisper-cpp":
			return "auto"
		default:
			// most providers use empty string or omit the parameter
			return ""
		}
	}

	switch providerName {
	case "deepgram":
		// deepgram prefers locale codes for some languages
		return toDeepgramFormat(code)
	default:
		// whisper-cpp, openai, groq, mistral, elevenlabs use standard codes
		return code
	}
}

// toDeepgramFormat maps standard codes to Deepgram's preferred format
func toDeepgramFormat(code string) string {
	// deepgram uses locale codes for English variants, standard for most others
	deepgramMappings := map[string]string{
		"en": "en-US",
		"es": "es",    // Spanish uses base code
		"pt": "pt-BR", // Portuguese defaults to Brazilian
		"zh": "zh-CN", // Chinese defaults to Simplified
	}

	if mapped, ok := deepgramMappings[code]; ok {
		return mapped
	}
	return code
}
