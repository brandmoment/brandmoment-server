package llm

import (
	"context"
	"errors"
	"testing"
)

// mockMicroClassifier implements MicroClassifier using a function field.
type mockMicroClassifier struct {
	classifyFn func(ctx context.Context, phrase string) (MicroResult, error)
}

func (m *mockMicroClassifier) Classify(ctx context.Context, phrase string) (MicroResult, error) {
	return m.classifyFn(ctx, phrase)
}

var _ MicroClassifier = (*mockMicroClassifier)(nil)

// makeMockParser builds a MultiStageParser backed by a mockChatClient.
// The mockChatClient must be configured before calling this.
func makeMockParser(client ChatClient) *MultiStageParser {
	return NewMultiStageParser(client)
}

// validRulesJSON is a syntactically and semantically valid rules array.
const validRulesJSON = `[{"type":"blocklist","config":{"domains":["evil.com"],"bundle_ids":[]}}]`

// TestTwoLevelParser_Parse covers all routing paths.
func TestTwoLevelParser_Parse(t *testing.T) {
	const marginFloor = 0.05
	const ruleTypeFloor = 0.10

	tests := []struct {
		name              string
		microResult       MicroResult
		microErr          error
		llmCallCount      int // expected LLM Complete calls
		llmResponses      []ChatResponse
		wantRoute         Route
		wantConfidence    ConfidenceStatus
		wantUsedLLM       bool
		wantLLMTotalCalls int // if >0, check result.LLM.TotalCalls
		wantErr           bool
	}{
		{
			name: "invalid high margin → early fail, no LLM",
			microResult: MicroResult{
				Intent: IntentInvalid,
				Top1:   0.95,
				Margin: 0.30, // ≥ marginFloor
				Scores: map[Intent]float64{
					IntentInvalid:   0.95,
					IntentBlocklist: 0.65,
					IntentAmbiguous: 0.50,
				},
			},
			llmCallCount:   0,
			wantRoute:      RouteMicroEarlyFail,
			wantConfidence: ConfidenceStatusFail,
			wantUsedLLM:    false,
		},
		{
			name: "invalid low margin → escalate with check (not early fail)",
			microResult: MicroResult{
				Intent: IntentInvalid,
				Top1:   0.60,
				Margin: 0.02, // < marginFloor
				Scores: map[Intent]float64{
					IntentInvalid:   0.60,
					IntentBlocklist: 0.58,
					IntentAmbiguous: 0.40,
				},
			},
			// LLM: stage1 (analyze), stage2 (extract), self_check parse+verify = 4 calls
			llmCallCount: 4,
			llmResponses: []ChatResponse{
				{Content: `{"count":1,"rules":[{"type":"blocklist","summary":"block evil.com"}]}`, InputTokens: 20, OutputTokens: 10},
				{Content: `{"domains":["evil.com"],"bundle_ids":[]}`, InputTokens: 15, OutputTokens: 8},
				{Content: validRulesJSON, InputTokens: 20, OutputTokens: 5},
				{Content: "YES\nLooks correct.", InputTokens: 25, OutputTokens: 3},
			},
			wantRoute:      RouteLLMWithCheck,
			wantConfidence: ConfidenceStatusOK,
			wantUsedLLM:    true,
		},
		{
			name: "ambiguous high margin → LLM with check (ambiguous always escalates)",
			microResult: MicroResult{
				Intent: IntentAmbiguous,
				Top1:   0.80,
				Margin: 0.50, // high margin but still escalates
				Scores: map[Intent]float64{
					IntentAmbiguous: 0.80,
					IntentBlocklist: 0.30,
					IntentInvalid:   0.20,
				},
			},
			// LLM: stage1 + stage2 + self_check (parse+verify) = 4 calls
			llmCallCount: 4,
			llmResponses: []ChatResponse{
				{Content: `{"count":1,"rules":[{"type":"blocklist","summary":"block evil.com"}]}`, InputTokens: 20, OutputTokens: 10},
				{Content: `{"domains":["evil.com"],"bundle_ids":[]}`, InputTokens: 15, OutputTokens: 8},
				{Content: validRulesJSON, InputTokens: 20, OutputTokens: 5},
				{Content: "NO\nMissing context.", InputTokens: 25, OutputTokens: 3},
			},
			wantRoute:      RouteLLMWithCheck,
			wantConfidence: ConfidenceStatusUnsure, // self-check said NO → downgraded
			wantUsedLLM:    true,
		},
		{
			name: "blocklist high margin → micro_answer (extract-only, 1 LLM call)",
			microResult: MicroResult{
				Intent: IntentBlocklist,
				Top1:   0.92,
				Margin: 0.25, // ≥ ruleTypeFloor
				Scores: map[Intent]float64{
					IntentBlocklist: 0.92,
					IntentAllowlist: 0.67,
					IntentAmbiguous: 0.40,
				},
			},
			// ExtractOnly: 1 LLM call (extract) only — no analyze, no self-check
			llmCallCount: 1,
			llmResponses: []ChatResponse{
				{Content: `{"domains":["evil.com"],"bundle_ids":[]}`, InputTokens: 15, OutputTokens: 8},
			},
			wantRoute:         RouteMicroAnswer,
			wantConfidence:    ConfidenceStatusOK,
			wantUsedLLM:       true,
			wantLLMTotalCalls: 1,
		},
		{
			name: "blocklist low margin → LLM with check (below ruleTypeFloor)",
			microResult: MicroResult{
				Intent: IntentBlocklist,
				Top1:   0.65,
				Margin: 0.04, // < ruleTypeFloor (0.10) but ≥ marginFloor (0.05)? No: 0.04 < 0.05
				// Actually 0.04 < marginFloor=0.05 and < ruleTypeFloor=0.10 → goes to llm_with_check
				Scores: map[Intent]float64{
					IntentBlocklist: 0.65,
					IntentAllowlist: 0.61,
					IntentAmbiguous: 0.40,
				},
			},
			// LLM: stage1 + stage2 + self_check (parse+verify) = 4 calls
			llmCallCount: 4,
			llmResponses: []ChatResponse{
				{Content: `{"count":1,"rules":[{"type":"blocklist","summary":"block evil.com"}]}`, InputTokens: 20, OutputTokens: 10},
				{Content: `{"domains":["evil.com"],"bundle_ids":[]}`, InputTokens: 15, OutputTokens: 8},
				{Content: validRulesJSON, InputTokens: 20, OutputTokens: 5},
				{Content: "YES\nAll good.", InputTokens: 25, OutputTokens: 3},
			},
			wantRoute:      RouteLLMWithCheck,
			wantConfidence: ConfidenceStatusOK,
			wantUsedLLM:    true,
		},
		{
			name:     "micro classifier returns error → error propagated",
			microErr: errors.New("embedding service unavailable"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build mock micro.
			microMock := &mockMicroClassifier{
				classifyFn: func(ctx context.Context, phrase string) (MicroResult, error) {
					if tt.microErr != nil {
						return MicroResult{}, tt.microErr
					}
					return tt.microResult, nil
				},
			}

			// Build mock LLM client.
			callIndex := 0
			llmMock := &mockChatClient{
				provider: ProviderOpenAI,
				completeFn: func(ctx context.Context, req ChatRequest) (ChatResponse, error) {
					if callIndex >= len(tt.llmResponses) {
						t.Errorf("unexpected LLM call #%d (only %d responses configured)", callIndex+1, len(tt.llmResponses))
						return ChatResponse{}, errors.New("no response configured")
					}
					resp := tt.llmResponses[callIndex]
					callIndex++
					return resp, nil
				},
			}

			parser := NewTwoLevelParser(microMock, makeMockParser(llmMock), marginFloor, ruleTypeFloor)
			result, err := parser.Parse(context.Background(), "block evil.com")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Route != tt.wantRoute {
				t.Errorf("Route = %q, want %q", result.Route, tt.wantRoute)
			}

			if result.Confidence != tt.wantConfidence {
				t.Errorf("Confidence = %q, want %q", result.Confidence, tt.wantConfidence)
			}

			if result.UsedLLM != tt.wantUsedLLM {
				t.Errorf("UsedLLM = %v, want %v", result.UsedLLM, tt.wantUsedLLM)
			}

			if tt.wantRoute == RouteMicroEarlyFail {
				if result.LLM != nil {
					t.Error("LLM result should be nil on early fail")
				}
				if result.RulesJSON != "" {
					t.Errorf("RulesJSON should be empty on early fail, got %q", result.RulesJSON)
				}
			} else {
				if result.LLM == nil {
					t.Error("LLM result should not be nil when LLM was used")
				}
			}

			if tt.wantLLMTotalCalls > 0 {
				if result.LLM == nil {
					t.Error("LLM result is nil, cannot check TotalCalls")
				} else if result.LLM.TotalCalls != tt.wantLLMTotalCalls {
					t.Errorf("LLM.TotalCalls = %d, want %d", result.LLM.TotalCalls, tt.wantLLMTotalCalls)
				}
			}

			if callIndex != tt.llmCallCount {
				t.Errorf("LLM calls = %d, want %d", callIndex, tt.llmCallCount)
			}
		})
	}
}

func TestTwoLevelParser_MicroError_Propagated(t *testing.T) {
	wantErr := errors.New("network failure")
	microMock := &mockMicroClassifier{
		classifyFn: func(ctx context.Context, phrase string) (MicroResult, error) {
			return MicroResult{}, wantErr
		},
	}

	llmMock := &mockChatClient{
		provider: ProviderOpenAI,
		completeFn: func(ctx context.Context, req ChatRequest) (ChatResponse, error) {
			t.Error("LLM should not be called when micro fails")
			return ChatResponse{}, nil
		},
	}

	parser := NewTwoLevelParser(microMock, makeMockParser(llmMock), 0.05, 0.10)
	_, err := parser.Parse(context.Background(), "some phrase")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error chain: got %v, want to contain %v", err, wantErr)
	}
}
