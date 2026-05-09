package llm

import "testing"

func TestBuildOpenAIChatRequest_OmitsTemperatureForRestrictedModels(t *testing.T) {
	tests := []struct {
		model        string
		wantTemp     float32
		wantTempZero bool
	}{
		// Pre-existing GPT-4o LLM models accept arbitrary temperature.
		{model: "gpt-4o-mini", wantTemp: 0.3, wantTempZero: false},
		{model: "gpt-4o", wantTemp: 0.3, wantTempZero: false},

		// GPT-5 family rejects sampling params; adapter must leave Temperature unset.
		{model: "gpt-5", wantTemp: 0, wantTempZero: true},
		{model: "gpt-5-mini", wantTemp: 0, wantTempZero: true},
		{model: "gpt-5.4", wantTemp: 0, wantTempZero: true},
		{model: "gpt-5.4-mini", wantTemp: 0, wantTempZero: true},
		{model: "gpt-5.4-nano", wantTemp: 0, wantTempZero: true},

		// Unknown models fall through to the default behavior (Temperature=0.3)
		// rather than being silently neutered. Lets users opt into preview models
		// not yet in the registry without losing cleanup quality.
		{model: "gpt-4o-mini-2030-future-preview", wantTemp: 0.3, wantTempZero: false},
	}

	for _, tc := range tests {
		t.Run(tc.model, func(t *testing.T) {
			req := buildOpenAIChatRequest(tc.model, "system", "user")

			if req.Model != tc.model {
				t.Errorf("Model = %q, want %q", req.Model, tc.model)
			}
			if len(req.Messages) != 2 {
				t.Fatalf("Messages len = %d, want 2", len(req.Messages))
			}

			if tc.wantTempZero && req.Temperature != 0 {
				t.Errorf("Temperature = %v, want unset (0) for restricted model", req.Temperature)
			}
			if !tc.wantTempZero && req.Temperature != tc.wantTemp {
				t.Errorf("Temperature = %v, want %v", req.Temperature, tc.wantTemp)
			}
		})
	}
}
