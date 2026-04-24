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

	tests := []struct {
		name           string
		microResult    MicroResult
		microErr       error
		llmCallCount   int // expected LLM Complete calls
		llmResponses   []ChatResponse
		wantRoute      Route
		wantConfidence ConfidenceStatus
		wantUsedLLM    bool
		wantErr        bool
	}{
		{
			name: "gibberish high margin → early fail, no LLM",
			microResult: MicroResult{
				Intent: IntentGibberish,
				Top1:   0.95,
				Margin: 0.30,
				Scores: map[Intent]float64{
					IntentGibberish: 0.95,
					IntentValid:     0.65,
					IntentAmbiguous: 0.50,
				},
			},
			llmCallCount:   0,
			wantRoute:      RouteMicroEarlyFail,
			wantConfidence: ConfidenceStatusFail,
			wantUsedLLM:    false,
		},
		{
			name: "gibberish low margin → escalate with check (not early fail)",
			microResult: MicroResult{
				Intent: IntentGibberish,
				Top1:   0.60,
				Margin: 0.02, // below marginFloor
				Scores: map[Intent]float64{
					IntentGibberish: 0.60,
					IntentValid:     0.58,
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
			name: "valid high margin → LLM direct (no self-check)",
			microResult: MicroResult{
				Intent: IntentValid,
				Top1:   0.90,
				Margin: 0.20,
				Scores: map[Intent]float64{
					IntentValid:     0.90,
					IntentAmbiguous: 0.70,
					IntentGibberish: 0.40,
				},
			},
			// LLM: stage1 + stage2 only = 2 calls (no self-check)
			llmCallCount: 2,
			llmResponses: []ChatResponse{
				{Content: `{"count":1,"rules":[{"type":"blocklist","summary":"block evil.com"}]}`, InputTokens: 20, OutputTokens: 10},
				{Content: `{"domains":["evil.com"],"bundle_ids":[]}`, InputTokens: 15, OutputTokens: 8},
			},
			wantRoute:      RouteLLMDirect,
			wantConfidence: ConfidenceStatusOK,
			wantUsedLLM:    true,
		},
		{
			name: "ambiguous → LLM with check",
			microResult: MicroResult{
				Intent: IntentAmbiguous,
				Top1:   0.80,
				Margin: 0.15,
				Scores: map[Intent]float64{
					IntentAmbiguous: 0.80,
					IntentValid:     0.65,
					IntentGibberish: 0.40,
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
			name: "valid low margin → LLM with check",
			microResult: MicroResult{
				Intent: IntentValid,
				Top1:   0.60,
				Margin: 0.02, // below marginFloor
				Scores: map[Intent]float64{
					IntentValid:     0.60,
					IntentAmbiguous: 0.58,
					IntentGibberish: 0.40,
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

			parser := NewTwoLevelParser(microMock, makeMockParser(llmMock), marginFloor)
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

	parser := NewTwoLevelParser(microMock, makeMockParser(llmMock), 0.05)
	_, err := parser.Parse(context.Background(), "some phrase")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error chain: got %v, want to contain %v", err, wantErr)
	}
}
