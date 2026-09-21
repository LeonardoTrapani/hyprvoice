package llm

import "testing"

func TestBuildOpenAIChatRequest(t *testing.T) {
	tests := []struct {
		name               string
		model              string
		omitSamplingParams bool
		wantTemp           float32 // 0 means Temperature must be left unset
	}{
		{
			name:               "gpt-4o-mini accepts sampling params",
			model:              "gpt-4o-mini",
			omitSamplingParams: false,
			wantTemp:           0.3,
		},
		{
			name:               "gpt-4o accepts sampling params",
			model:              "gpt-4o",
			omitSamplingParams: false,
			wantTemp:           0.3,
		},
		{
			name:               "gpt-5.4-mini omits sampling params",
			model:              "gpt-5.4-mini",
			omitSamplingParams: true,
			wantTemp:           0,
		},
		{
			name:               "unknown model defaults to sampling params on",
			model:              "gpt-4o-mini-2030-future-preview",
			omitSamplingParams: false,
			wantTemp:           0.3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := buildOpenAIChatRequest(tc.model, "system", "user", tc.omitSamplingParams)

			if req.Model != tc.model {
				t.Errorf("Model = %q, want %q", req.Model, tc.model)
			}
			if len(req.Messages) != 2 {
				t.Fatalf("Messages len = %d, want 2", len(req.Messages))
			}
			if req.Temperature != tc.wantTemp {
				t.Errorf("Temperature = %v, want %v", req.Temperature, tc.wantTemp)
			}
		})
	}
}

func TestNewOpenAIAdapter_ResolvesRestrictedSampling(t *testing.T) {
	tests := []struct {
		model    string
		wantOmit bool
	}{
		// Pre-existing GPT-4o LLM models accept sampling params.
		{model: "gpt-4o-mini", wantOmit: false},
		{model: "gpt-4o", wantOmit: false},

		// GPT-5 family rejects sampling params; adapter must cache the flag.
		{model: "gpt-5", wantOmit: true},
		{model: "gpt-5-mini", wantOmit: true},
		{model: "gpt-5.4", wantOmit: true},
		{model: "gpt-5.4-mini", wantOmit: true},
		{model: "gpt-5.4-nano", wantOmit: true},

		// Models not in the registry fall through to the default behavior.
		{model: "", wantOmit: false},
		{model: "gpt-4o-mini-2030-future-preview", wantOmit: false},
	}

	for _, tc := range tests {
		t.Run(tc.model, func(t *testing.T) {
			a := NewOpenAIAdapter(Config{Model: tc.model, APIKey: "test"})
			if a.omitSamplingParams != tc.wantOmit {
				t.Errorf("omitSamplingParams = %v, want %v", a.omitSamplingParams, tc.wantOmit)
			}
		})
	}
}
