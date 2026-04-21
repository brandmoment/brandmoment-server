package llm

import (
	"context"
	"errors"
	"testing"
)

// mockChatClient implements ChatClient using function fields.
type mockChatClient struct {
	completeFn func(ctx context.Context, req ChatRequest) (ChatResponse, error)
	provider   Provider
}

func (m *mockChatClient) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	return m.completeFn(ctx, req)
}

func (m *mockChatClient) Provider() Provider {
	return m.provider
}

// compile-time check
var _ ChatClient = (*mockChatClient)(nil)

func TestParseSelfCheckResponse(t *testing.T) {
	tests := []struct {
		name            string
		raw             string
		wantYes         bool
		wantExplanation string
	}{
		{
			name:            "YES with explanation",
			raw:             "YES\nThe JSON captures all intent correctly.",
			wantYes:         true,
			wantExplanation: "The JSON captures all intent correctly.",
		},
		{
			name:            "NO with explanation",
			raw:             "NO\nMissing geo restriction.",
			wantYes:         false,
			wantExplanation: "Missing geo restriction.",
		},
		{
			name:            "YES lowercase",
			raw:             "yes\nLooks good.",
			wantYes:         true,
			wantExplanation: "Looks good.",
		},
		{
			name:            "YES with trailing spaces",
			raw:             "  YES  \nAll covered.",
			wantYes:         true,
			wantExplanation: "All covered.",
		},
		{
			name:            "NO no explanation",
			raw:             "NO",
			wantYes:         false,
			wantExplanation: "",
		},
		{
			name:            "unparseable verdict treated as NO",
			raw:             "MAYBE\nNot sure.",
			wantYes:         false,
			wantExplanation: "Not sure.",
		},
		{
			name:            "empty response",
			raw:             "",
			wantYes:         false,
			wantExplanation: "",
		},
		{
			name:            "YES with multiline explanation",
			raw:             "YES\nLine one.\nLine two.",
			wantYes:         true,
			wantExplanation: "Line one.\nLine two.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotYes, gotExp := parseSelfCheckResponse(tt.raw)
			if gotYes != tt.wantYes {
				t.Errorf("yes = %v, want %v", gotYes, tt.wantYes)
			}
			if gotExp != tt.wantExplanation {
				t.Errorf("explanation = %q, want %q", gotExp, tt.wantExplanation)
			}
		})
	}
}

func TestCheckSelfCheck_YesVerdict(t *testing.T) {
	validRulesJSON := `[{"type":"blocklist","config":{"domains":["evil.com"]}}]`
	callCount := 0
	client := &mockChatClient{
		provider: ProviderOpenAI,
		completeFn: func(ctx context.Context, req ChatRequest) (ChatResponse, error) {
			callCount++
			switch callCount {
			case 1:
				// parse call
				return ChatResponse{Content: validRulesJSON, InputTokens: 10, OutputTokens: 5}, nil
			case 2:
				// verify call
				return ChatResponse{Content: "YES\nPerfectly captured.", InputTokens: 20, OutputTokens: 3}, nil
			default:
				return ChatResponse{}, errors.New("unexpected call")
			}
		},
	}

	rulesJSON, result := CheckSelfCheck(context.Background(), client, "block evil.com")

	if rulesJSON != validRulesJSON {
		t.Errorf("rulesJSON = %q, want %q", rulesJSON, validRulesJSON)
	}
	if result.Status != ConfidenceStatusOK {
		t.Errorf("Status = %q, want OK", result.Status)
	}
	if result.Explanation != "Perfectly captured." {
		t.Errorf("Explanation = %q, want %q", result.Explanation, "Perfectly captured.")
	}
	if result.InputTokens != 30 {
		t.Errorf("InputTokens = %d, want 30", result.InputTokens)
	}
	if result.OutputTokens != 8 {
		t.Errorf("OutputTokens = %d, want 8", result.OutputTokens)
	}
	if callCount != 2 {
		t.Errorf("expected 2 LLM calls, got %d", callCount)
	}
}

func TestCheckSelfCheck_NoVerdict(t *testing.T) {
	validRulesJSON := `[{"type":"geo_filter","config":{"mode":"include","country_codes":["US"]}}]`
	callCount := 0
	client := &mockChatClient{
		provider: ProviderOpenAI,
		completeFn: func(ctx context.Context, req ChatRequest) (ChatResponse, error) {
			callCount++
			switch callCount {
			case 1:
				return ChatResponse{Content: validRulesJSON, InputTokens: 15, OutputTokens: 8}, nil
			case 2:
				return ChatResponse{Content: "NO\nMissing frequency cap constraint.", InputTokens: 25, OutputTokens: 6}, nil
			default:
				return ChatResponse{}, errors.New("unexpected call")
			}
		},
	}

	rulesJSON, result := CheckSelfCheck(context.Background(), client, "only US with frequency cap")

	if rulesJSON != validRulesJSON {
		t.Errorf("rulesJSON = %q, want %q", rulesJSON, validRulesJSON)
	}
	if result.Status != ConfidenceStatusUnsure {
		t.Errorf("Status = %q, want UNSURE", result.Status)
	}
	if result.Explanation != "Missing frequency cap constraint." {
		t.Errorf("Explanation = %q", result.Explanation)
	}
}

func TestCheckSelfCheck_ParseCallFails(t *testing.T) {
	client := &mockChatClient{
		provider: ProviderOpenAI,
		completeFn: func(ctx context.Context, req ChatRequest) (ChatResponse, error) {
			return ChatResponse{}, errors.New("network error")
		},
	}

	rulesJSON, result := CheckSelfCheck(context.Background(), client, "some phrase")

	if rulesJSON != "" {
		t.Errorf("rulesJSON = %q, want empty", rulesJSON)
	}
	if result.Status != ConfidenceStatusFail {
		t.Errorf("Status = %q, want FAIL", result.Status)
	}
}

func TestCheckSelfCheck_VerifyCallFails(t *testing.T) {
	validRulesJSON := `[{"type":"allowlist","config":{"domains":["good.com"]}}]`
	callCount := 0
	client := &mockChatClient{
		provider: ProviderOpenAI,
		completeFn: func(ctx context.Context, req ChatRequest) (ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return ChatResponse{Content: validRulesJSON, InputTokens: 10, OutputTokens: 5}, nil
			}
			return ChatResponse{}, errors.New("verify API timeout")
		},
	}

	rulesJSON, result := CheckSelfCheck(context.Background(), client, "allow only good.com")

	if rulesJSON != validRulesJSON {
		t.Errorf("rulesJSON = %q, want %q", rulesJSON, validRulesJSON)
	}
	if result.Status != ConfidenceStatusUnsure {
		t.Errorf("Status = %q, want UNSURE when verify fails", result.Status)
	}
	if result.Explanation != "verify call failed" {
		t.Errorf("Explanation = %q, want 'verify call failed'", result.Explanation)
	}
}

func TestCheckSelfCheck_UnparseableVerifyResponse(t *testing.T) {
	validRulesJSON := `[{"type":"blocklist","config":{"domains":["x.com"]}}]`
	callCount := 0
	client := &mockChatClient{
		provider: ProviderOpenAI,
		completeFn: func(ctx context.Context, req ChatRequest) (ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return ChatResponse{Content: validRulesJSON, InputTokens: 10, OutputTokens: 4}, nil
			}
			// Returns something that is not YES
			return ChatResponse{Content: "UNCLEAR\nI cannot determine.", InputTokens: 18, OutputTokens: 4}, nil
		},
	}

	_, result := CheckSelfCheck(context.Background(), client, "block x.com")

	// Non-YES response should result in UNSURE
	if result.Status != ConfidenceStatusUnsure {
		t.Errorf("Status = %q, want UNSURE for non-YES response", result.Status)
	}
}

func TestCheckSelfCheck_LatencyPopulated(t *testing.T) {
	client := &mockChatClient{
		provider: ProviderOpenAI,
		completeFn: func(ctx context.Context, req ChatRequest) (ChatResponse, error) {
			return ChatResponse{Content: `[{"type":"blocklist","config":{"domains":["a.com"]}}]`}, nil
		},
	}

	_, result := CheckSelfCheck(context.Background(), client, "block a.com")
	if result.Latency <= 0 {
		t.Error("expected Latency > 0")
	}
}
