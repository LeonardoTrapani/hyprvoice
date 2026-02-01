package provider

var openaiTranscriptionLanguages = []string{
	"af", "ar", "hy", "az", "be", "bs", "bg", "ca", "zh", "hr", "cs", "da",
	"nl", "en", "et", "fi", "fr", "gl", "de", "el", "he", "hi", "hu", "is",
	"id", "it", "ja", "kn", "kk", "ko", "lv", "lt", "mk", "ms", "mr", "mi",
	"ne", "no", "fa", "pl", "pt", "ro", "ru", "sr", "sk", "sl", "es", "sw",
	"sv", "tl", "ta", "th", "tr", "uk", "ur", "vi", "cy",
}

var groqTranscriptionLanguages = openaiTranscriptionLanguages
var mistralTranscriptionLanguages = openaiTranscriptionLanguages
var whisperTranscriptionLanguages = openaiTranscriptionLanguages

var whisperEnglishOnlyLanguages = []string{"en"}

var deepgramNova3Languages = []string{
	"multi",
	"ar", "ar-AE", "ar-SA", "ar-QA", "ar-KW", "ar-SY", "ar-LB", "ar-PS", "ar-JO", "ar-EG", "ar-SD", "ar-TD", "ar-MA", "ar-DZ", "ar-TN", "ar-IQ", "ar-IR",
	"be", "bn", "bs", "bg", "ca", "hr", "cs", "da", "da-DK", "nl", "nl-BE",
	"en", "en-US", "en-AU", "en-GB", "en-IN", "en-NZ", "et", "fi", "fr", "fr-CA",
	"de", "de-CH", "el", "hi", "hu", "id", "it", "ja", "kn", "ko", "ko-KR",
	"lv", "lt", "mk", "ms", "mr", "no", "pl", "pt", "pt-BR", "pt-PT", "ro",
	"ru", "sr", "sk", "sl", "es", "es-419", "sv", "sv-SE", "tl", "ta", "te",
	"tr", "uk", "vi",
}

var deepgramNova2Languages = []string{
	"multi",
	"bg", "ca", "zh", "zh-CN", "zh-Hans", "zh-TW", "zh-Hant", "zh-HK", "cs",
	"da", "da-DK", "nl", "nl-BE", "en", "en-US", "en-AU", "en-GB", "en-NZ", "en-IN",
	"et", "fi", "fr", "fr-CA", "de", "de-CH", "el", "hi", "hu", "id", "it", "ja",
	"ko", "ko-KR", "lv", "lt", "ms", "no", "pl", "pt", "pt-BR", "pt-PT", "ro",
	"ru", "sk", "es", "es-419", "sv", "sv-SE", "th", "th-TH", "tr", "uk", "vi",
}

var elevenLabsTranscriptionLanguages = []string{
	"bel", "bos", "bul", "cat", "hrv", "ces", "dan", "nld", "eng", "est", "fin", "fra",
	"glg", "deu", "ell", "hun", "isl", "ind", "ita", "jpn", "kan", "lav", "mkd", "msa",
	"mal", "nor", "pol", "por", "ron", "rus", "slk", "spa", "swe", "tur", "ukr", "vie",
	"hye", "aze", "ben", "yue", "fil", "kat", "guj", "hin", "kaz", "lit", "mlt", "cmn",
	"mar", "nep", "ori", "fas", "srp", "slv", "swa", "tam", "tel",
	"afr", "ara", "asm", "ast", "mya", "hau", "heb", "jav", "kor", "kir", "ltz", "mri",
	"oci", "pan", "tgk", "tha", "uzb", "cym",
	"amh", "lug", "ibo", "gle", "khm", "kur", "lao", "mon", "nso", "pus", "sna", "snd",
	"som", "urd", "wol", "xho", "yor", "zul",
}
