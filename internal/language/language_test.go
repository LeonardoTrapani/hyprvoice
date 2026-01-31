package language

import "testing"

func TestFromCode(t *testing.T) {
	tests := []struct {
		code     string
		wantCode string
		wantName string
	}{
		{"en", "en", "English"},
		{"es", "es", "Spanish"},
		{"zh", "zh", "Chinese"},
		{"invalid", "", "Auto-detect"},
		{"", "", "Auto-detect"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := FromCode(tt.code)
			if got.Code != tt.wantCode {
				t.Errorf("FromCode(%q).Code = %q, want %q", tt.code, got.Code, tt.wantCode)
			}
			if got.Name != tt.wantName {
				t.Errorf("FromCode(%q).Name = %q, want %q", tt.code, got.Name, tt.wantName)
			}
		})
	}
}

func TestFromCodeEnglish(t *testing.T) {
	lang := FromCode("en")
	if lang.Code != "en" {
		t.Errorf("FromCode('en').Code = %q, want 'en'", lang.Code)
	}
	if lang.Name != "English" {
		t.Errorf("FromCode('en').Name = %q, want 'English'", lang.Name)
	}
	if lang.NativeName != "English" {
		t.Errorf("FromCode('en').NativeName = %q, want 'English'", lang.NativeName)
	}
}

func TestIsValidCode(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"en", true},
		{"es", true},
		{"zh", true},
		{"invalid", false},
		{"", true}, // auto is valid
		{"xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := IsValidCode(tt.code)
			if got != tt.want {
				t.Errorf("IsValidCode(%q) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestList(t *testing.T) {
	list := List()
	if len(list) != 57 {
		t.Errorf("List() returned %d languages, want 57", len(list))
	}

	// verify English is in the list
	found := false
	for _, lang := range list {
		if lang.Code == "en" {
			found = true
			break
		}
	}
	if !found {
		t.Error("List() does not contain English")
	}
}

func TestCodes(t *testing.T) {
	codes := Codes()
	if len(codes) != 57 {
		t.Errorf("Codes() returned %d codes, want 57", len(codes))
	}

	// verify 'en' is in the codes
	found := false
	for _, code := range codes {
		if code == "en" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Codes() does not contain 'en'")
	}
}

func TestAllLanguageCodes(t *testing.T) {
	codes := AllLanguageCodes()
	if len(codes) != 57 {
		t.Errorf("AllLanguageCodes() returned %d codes, want 57", len(codes))
	}
}

func TestAuto(t *testing.T) {
	if Auto.Code != "" {
		t.Errorf("Auto.Code = %q, want empty string", Auto.Code)
	}
	if Auto.Name != "Auto-detect" {
		t.Errorf("Auto.Name = %q, want 'Auto-detect'", Auto.Name)
	}
}

func TestToProviderFormat(t *testing.T) {
	tests := []struct {
		code     string
		provider string
		want     string
	}{
		// whisper-cpp
		{"en", "whisper-cpp", "en"},
		{"es", "whisper-cpp", "es"},
		{"", "whisper-cpp", "auto"},

		// openai
		{"en", "openai", "en"},
		{"", "openai", ""},

		// groq (openai-compatible)
		{"en", "groq", "en"},
		{"", "groq", ""},

		// mistral (openai-compatible)
		{"en", "mistral", "en"},
		{"", "mistral", ""},

		// deepgram (uses locale codes)
		{"en", "deepgram", "en-US"},
		{"es", "deepgram", "es"},
		{"pt", "deepgram", "pt-BR"},
		{"zh", "deepgram", "zh-CN"},
		{"fr", "deepgram", "fr"}, // no special mapping, passthrough
		{"", "deepgram", ""},

		// elevenlabs
		{"en", "elevenlabs", "en"},
		{"", "elevenlabs", ""},
	}

	for _, tt := range tests {
		t.Run(tt.code+"_"+tt.provider, func(t *testing.T) {
			got := ToProviderFormat(tt.code, tt.provider)
			if got != tt.want {
				t.Errorf("ToProviderFormat(%q, %q) = %q, want %q", tt.code, tt.provider, got, tt.want)
			}
		})
	}
}
