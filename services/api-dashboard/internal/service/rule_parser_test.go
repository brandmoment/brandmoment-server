package service

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/llm"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

// mockLLMClient implements llm.ChatClient for service-level tests.
type mockLLMClient struct {
	completeFn func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error)
}

func (m *mockLLMClient) Complete(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	return m.completeFn(ctx, req)
}

func (m *mockLLMClient) Provider() llm.Provider {
	return llm.ProviderOpenAI
}

var _ llm.ChatClient = (*mockLLMClient)(nil)

func newTestRuleParserService(client llm.ChatClient) *RuleParserService {
	return NewRuleParserService(client, noop.NewTracerProvider())
}

// buildSingleCallClient returns a client that always returns the given JSON on every Complete call.
func buildSingleCallClient(content string) *mockLLMClient {
	return &mockLLMClient{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: content, InputTokens: 10, OutputTokens: 5}, nil
		},
	}
}

// buildErrorClient returns a client that always returns an error.
func buildErrorClient(err error) *mockLLMClient {
	return &mockLLMClient{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{}, err
		},
	}
}

// ---- resolveApproaches ----

func TestResolveApproaches_EmptyUsesDefaults(t *testing.T) {
	result := resolveApproaches(nil)
	if len(result) != 2 {
		t.Fatalf("expected 2 default approaches, got %d", len(result))
	}
}

func TestResolveApproaches_AllInvalidFallsBack(t *testing.T) {
	result := resolveApproaches([]string{"bogus", "nonsense"})
	if len(result) != 2 {
		t.Fatalf("expected 2 default approaches for all-invalid input, got %d", len(result))
	}
}

func TestResolveApproaches_SingleConstraint(t *testing.T) {
	result := resolveApproaches([]string{"constraint"})
	if len(result) != 1 || result[0] != llm.ApproachConstraint {
		t.Errorf("expected [constraint], got %v", result)
	}
}

func TestResolveApproaches_SingleSelfCheck(t *testing.T) {
	result := resolveApproaches([]string{"self_check"})
	if len(result) != 1 || result[0] != llm.ApproachSelfCheck {
		t.Errorf("expected [self_check], got %v", result)
	}
}

func TestResolveApproaches_Deduplication(t *testing.T) {
	result := resolveApproaches([]string{"constraint", "constraint", "self_check"})
	if len(result) != 2 {
		t.Errorf("expected 2 deduplicated approaches, got %d: %v", len(result), result)
	}
}

// ---- overallVerdict ----

func TestOverallVerdict(t *testing.T) {
	tests := []struct {
		name       string
		approaches map[llm.ApproachName]ApproachReport
		want       llm.ConfidenceStatus
	}{
		{
			name: "all OK returns OK",
			approaches: map[llm.ApproachName]ApproachReport{
				llm.ApproachConstraint: {Status: llm.ConfidenceStatusOK},
				llm.ApproachSelfCheck:  {Status: llm.ConfidenceStatusOK},
			},
			want: llm.ConfidenceStatusOK,
		},
		{
			name: "one UNSURE returns UNSURE",
			approaches: map[llm.ApproachName]ApproachReport{
				llm.ApproachConstraint: {Status: llm.ConfidenceStatusOK},
				llm.ApproachSelfCheck:  {Status: llm.ConfidenceStatusUnsure},
			},
			want: llm.ConfidenceStatusUnsure,
		},
		{
			name: "one FAIL dominates",
			approaches: map[llm.ApproachName]ApproachReport{
				llm.ApproachConstraint: {Status: llm.ConfidenceStatusUnsure},
				llm.ApproachSelfCheck:  {Status: llm.ConfidenceStatusFail},
			},
			want: llm.ConfidenceStatusFail,
		},
		{
			name: "empty map returns OK",
			approaches: map[llm.ApproachName]ApproachReport{},
			want:       llm.ConfidenceStatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := overallVerdict(tt.approaches)
			if got != tt.want {
				t.Errorf("overallVerdict = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---- RuleParserService.Parse ----

func TestRuleParserService_Parse_EmptyPhrase(t *testing.T) {
	svc := newTestRuleParserService(buildSingleCallClient(`[]`))

	_, _, err := svc.Parse(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty phrase")
	}
	if !errors.Is(err, model.ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput", err)
	}
}

func TestRuleParserService_Parse_WhitespaceOnlyPhrase(t *testing.T) {
	svc := newTestRuleParserService(buildSingleCallClient(`[]`))

	_, _, err := svc.Parse(context.Background(), "   ", nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only phrase")
	}
	if !errors.Is(err, model.ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput", err)
	}
}

func TestRuleParserService_Parse_ConstraintOnly_ValidRules(t *testing.T) {
	validJSON := `[{"type":"blocklist","config":{"domains":["evil.com"]}}]`
	svc := newTestRuleParserService(buildSingleCallClient(validJSON))

	rules, report, err := svc.Parse(context.Background(), "block evil.com", []string{"constraint"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Type != "blocklist" {
		t.Errorf("rule type = %q, want blocklist", rules[0].Type)
	}
	if report.Overall != llm.ConfidenceStatusOK {
		t.Errorf("Overall = %q, want OK", report.Overall)
	}
	if _, ok := report.Approaches[llm.ApproachConstraint]; !ok {
		t.Error("expected constraint approach in report")
	}
}

func TestRuleParserService_Parse_ConstraintOnly_InvalidRules(t *testing.T) {
	// frequency_cap with 0 max_impressions will fail constraint check
	invalidJSON := `[{"type":"frequency_cap","config":{"max_impressions":0,"window_seconds":0}}]`
	svc := newTestRuleParserService(buildSingleCallClient(invalidJSON))

	_, report, err := svc.Parse(context.Background(), "some cap", []string{"constraint"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Overall != llm.ConfidenceStatusFail {
		t.Errorf("Overall = %q, want FAIL", report.Overall)
	}
	ar := report.Approaches[llm.ApproachConstraint]
	if ar.Status != llm.ConfidenceStatusFail {
		t.Errorf("constraint status = %q, want FAIL", ar.Status)
	}
}

func TestRuleParserService_Parse_SelfCheckOnly_YesVerdict(t *testing.T) {
	validJSON := `[{"type":"geo_filter","config":{"mode":"include","country_codes":["US"]}}]`
	callCount := 0
	client := &mockLLMClient{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return llm.ChatResponse{Content: validJSON, InputTokens: 10, OutputTokens: 5}, nil
			}
			return llm.ChatResponse{Content: "YES\nCaptures the US geo filter perfectly.", InputTokens: 20, OutputTokens: 5}, nil
		},
	}
	svc := newTestRuleParserService(client)

	rules, report, err := svc.Parse(context.Background(), "only in USA", []string{"self_check"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if report.Overall != llm.ConfidenceStatusOK {
		t.Errorf("Overall = %q, want OK", report.Overall)
	}
	ar := report.Approaches[llm.ApproachSelfCheck]
	if ar.Status != llm.ConfidenceStatusOK {
		t.Errorf("self_check status = %q, want OK", ar.Status)
	}
}

func TestRuleParserService_Parse_SelfCheckOnly_NoVerdict(t *testing.T) {
	validJSON := `[{"type":"platform_filter","config":{"mode":"include","platforms":["ios"]}}]`
	callCount := 0
	client := &mockLLMClient{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return llm.ChatResponse{Content: validJSON}, nil
			}
			return llm.ChatResponse{Content: "NO\nMissing android platform."}, nil
		},
	}
	svc := newTestRuleParserService(client)

	_, report, err := svc.Parse(context.Background(), "ios and android only", []string{"self_check"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Overall != llm.ConfidenceStatusUnsure {
		t.Errorf("Overall = %q, want UNSURE", report.Overall)
	}
}

func TestRuleParserService_Parse_BothApproaches_DefaultsToAll(t *testing.T) {
	validJSON := `[{"type":"frequency_cap","config":{"max_impressions":3,"window_seconds":86400}}]`
	callCount := 0
	client := &mockLLMClient{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				// self_check parse call
				return llm.ChatResponse{Content: validJSON, InputTokens: 10, OutputTokens: 5}, nil
			}
			if callCount == 2 {
				// self_check verify call
				return llm.ChatResponse{Content: "YES\nCorrect.", InputTokens: 15, OutputTokens: 3}, nil
			}
			// constraint will reuse rulesJSON from self_check, no extra LLM call needed
			return llm.ChatResponse{}, errors.New("unexpected extra call")
		},
	}
	svc := newTestRuleParserService(client)

	// passing nil defaults to both approaches
	_, report, err := svc.Parse(context.Background(), "cap at 3 per day", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := report.Approaches[llm.ApproachSelfCheck]; !ok {
		t.Error("expected self_check in approaches")
	}
	if _, ok := report.Approaches[llm.ApproachConstraint]; !ok {
		t.Error("expected constraint in approaches")
	}
}

func TestRuleParserService_Parse_LLMCallFails_ConstraintOnly(t *testing.T) {
	client := buildErrorClient(errors.New("LLM unavailable"))
	svc := newTestRuleParserService(client)

	_, report, err := svc.Parse(context.Background(), "block bad content", []string{"constraint"})
	// The parse itself should not error (approach failure is recorded in report)
	// but rules will be empty and rulesJSON will be empty.
	// Actually with empty rulesJSON, parseRulesJSON returns nil, nil — so no error.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ar := report.Approaches[llm.ApproachConstraint]
	if ar.Status != llm.ConfidenceStatusFail {
		t.Errorf("constraint status = %q, want FAIL", ar.Status)
	}
}

func TestRuleParserService_Parse_MultipleRules(t *testing.T) {
	multiJSON := `[
		{"type":"blocklist","config":{"domains":["evil.com"]}},
		{"type":"geo_filter","config":{"mode":"exclude","country_codes":["RU"]}},
		{"type":"frequency_cap","config":{"max_impressions":5,"window_seconds":3600}}
	]`
	svc := newTestRuleParserService(buildSingleCallClient(multiJSON))

	rules, report, err := svc.Parse(context.Background(), "block evil, no Russia, max 5/hour", []string{"constraint"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(rules))
	}
	if report.Overall != llm.ConfidenceStatusOK {
		t.Errorf("Overall = %q, want OK", report.Overall)
	}
}

func TestRuleParserService_Parse_InvalidApproachNames(t *testing.T) {
	// All invalid approach names should fall back to defaults (self_check + constraint)
	validJSON := `[{"type":"allowlist","config":{"domains":["ok.com"]}}]`
	callCount := 0
	client := &mockLLMClient{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			callCount++
			if callCount <= 2 {
				return llm.ChatResponse{Content: validJSON, InputTokens: 5, OutputTokens: 3}, nil
			}
			return llm.ChatResponse{Content: "YES\nCorrect."}, nil
		},
	}
	svc := newTestRuleParserService(client)

	_, report, err := svc.Parse(context.Background(), "allow ok.com", []string{"invalid_approach"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fall back to both defaults
	if len(report.Approaches) != 2 {
		t.Errorf("expected 2 approaches (defaults), got %d", len(report.Approaches))
	}
}
