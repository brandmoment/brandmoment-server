package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/llm"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

// ApproachReport holds the result of a single confidence-estimation method.
type ApproachReport struct {
	Status       llm.ConfidenceStatus `json:"status"`
	LatencyMS    string               `json:"latency_ms"`
	InputTokens  int                  `json:"input_tokens"`
	OutputTokens int                  `json:"output_tokens"`
	// Detail carries approach-specific metadata (reasoning, explanation, matches, errors…).
	Detail any `json:"detail,omitempty"`
}

// ConfidenceReport summarises all requested approaches and an overall verdict.
type ConfidenceReport struct {
	Overall    llm.ConfidenceStatus                `json:"overall"`
	Approaches map[llm.ApproachName]ApproachReport `json:"approaches"`
}

// RuleParserService translates natural-language phrases into validated PublisherRule
// objects and reports a confidence score via one or more estimation approaches.
type RuleParserService struct {
	client llm.ChatClient
	tracer trace.Tracer
}

// NewRuleParserService creates a RuleParserService using the given ChatClient.
func NewRuleParserService(client llm.ChatClient, tp trace.TracerProvider) *RuleParserService {
	return &RuleParserService{
		client: client,
		tracer: tp.Tracer("brandmoment/api-dashboard"),
	}
}

// Parse converts phrase into zero or more PublisherRule-compatible structs and
// runs the requested confidence approaches. approaches may contain any combination
// of "constraint", "self_check". If empty both run.
func (s *RuleParserService) Parse(
	ctx context.Context,
	phrase string,
	approaches []string,
) ([]model.PublisherRule, ConfidenceReport, error) {
	ctx, span := s.tracer.Start(ctx, "RuleParserService.Parse")
	defer span.End()

	if strings.TrimSpace(phrase) == "" {
		return nil, ConfidenceReport{}, fmt.Errorf("%w: phrase must not be empty", model.ErrInvalidInput)
	}

	active := resolveApproaches(approaches)

	slog.InfoContext(ctx, "parsing rule phrase",
		slog.String("phrase", phrase),
		slog.Any("approaches", active),
	)

	report := ConfidenceReport{
		Approaches: make(map[llm.ApproachName]ApproachReport, len(active)),
	}

	// rulesJSON accumulates the best available JSON from any approach that
	// performs a parse. We prefer whichever approach runs first.
	var rulesJSON string

	for _, approach := range active {
		switch approach {
		case llm.ApproachSelfCheck:
			rj, scr := llm.CheckSelfCheck(ctx, s.client, phrase)
			if rulesJSON == "" && rj != "" {
				rulesJSON = rj
			}
			report.Approaches[llm.ApproachSelfCheck] = ApproachReport{
				Status:       scr.Status,
				LatencyMS:    fmtLatencyMS(scr.Latency),
				InputTokens:  scr.InputTokens,
				OutputTokens: scr.OutputTokens,
				Detail:       map[string]any{"explanation": scr.Explanation},
			}

		case llm.ApproachConstraint:
			// Constraint needs a JSON string. If we don't have one yet, do a plain LLM parse.
			if rulesJSON == "" {
				rj, err := s.plainParse(ctx, phrase)
				if err != nil {
					report.Approaches[llm.ApproachConstraint] = ApproachReport{
						Status: llm.ConfidenceStatusFail,
						Detail: map[string]any{"error": err.Error()},
					}
					continue
				}
				rulesJSON = rj
			}
			cr := llm.CheckConstraint(ctx, rulesJSON)
			report.Approaches[llm.ApproachConstraint] = ApproachReport{
				Status:    cr.Status,
				LatencyMS: fmtLatencyMS(cr.Latency),
				Detail:    map[string]any{"errors": cr.Errors},
			}
		}
	}

	// Compute overall verdict: worst status across all approaches.
	report.Overall = overallVerdict(report.Approaches)

	// Decode rules from best available JSON.
	rules, err := parseRulesJSON(rulesJSON)
	if err != nil {
		span.RecordError(err)
		return nil, report, fmt.Errorf("parse rules: %w", err)
	}

	slog.InfoContext(ctx, "rule parse complete",
		slog.Int("rule_count", len(rules)),
		slog.String("overall_confidence", string(report.Overall)),
	)

	return rules, report, nil
}

// plainParse sends the base parse prompt and returns raw JSON.
func (s *RuleParserService) plainParse(ctx context.Context, phrase string) (string, error) {
	resp, err := s.client.Complete(ctx, llm.ChatRequest{
		Temperature: 0,
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: llm.SystemPrompt},
			{Role: llm.RoleUser, Content: phrase},
		},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Content), nil
}

// parseRulesJSON decodes a JSON array of {type, config} objects into model.PublisherRule slice.
func parseRulesJSON(raw string) ([]model.PublisherRule, error) {
	raw = llm.StripMarkdownFences(raw)
	if raw == "" {
		return nil, nil
	}
	type rawRule struct {
		Type   string          `json:"type"`
		Config json.RawMessage `json:"config"`
	}
	var parsed []rawRule
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, err
	}
	rules := make([]model.PublisherRule, 0, len(parsed))
	for _, r := range parsed {
		rules = append(rules, model.PublisherRule{
			Type:   r.Type,
			Config: r.Config,
		})
	}
	return rules, nil
}

// resolveApproaches returns the active set of approaches preserving order.
// Defaults to both if the input is empty or contains no valid names.
func resolveApproaches(requested []string) []llm.ApproachName {
	defaults := []llm.ApproachName{
		llm.ApproachSelfCheck,
		llm.ApproachConstraint,
	}
	if len(requested) == 0 {
		return defaults
	}
	valid := map[string]llm.ApproachName{
		string(llm.ApproachConstraint): llm.ApproachConstraint,
		string(llm.ApproachSelfCheck):  llm.ApproachSelfCheck,
	}
	out := make([]llm.ApproachName, 0, len(requested))
	seen := make(map[llm.ApproachName]bool)
	for _, r := range requested {
		if a, ok := valid[r]; ok && !seen[a] {
			out = append(out, a)
			seen[a] = true
		}
	}
	if len(out) == 0 {
		return defaults
	}
	return out
}

// overallVerdict returns the worst status across all approaches: FAIL > UNSURE > OK.
func overallVerdict(approaches map[llm.ApproachName]ApproachReport) llm.ConfidenceStatus {
	worst := llm.ConfidenceStatusOK
	for _, ar := range approaches {
		switch ar.Status {
		case llm.ConfidenceStatusFail:
			return llm.ConfidenceStatusFail
		case llm.ConfidenceStatusUnsure:
			worst = llm.ConfidenceStatusUnsure
		}
	}
	return worst
}

func fmtLatencyMS(d time.Duration) string {
	return fmt.Sprintf("%.1f", float64(d.Milliseconds()))
}
