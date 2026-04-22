package llm

import (
	"context"
	"log/slog"
	"time"
)

// RoutingDecision indicates which model produced the final result.
type RoutingDecision string

const (
	// RoutedPrimary means the primary (cheap) model had sufficient confidence.
	RoutedPrimary RoutingDecision = "primary"
	// RoutedFallback means the primary model's confidence was too low and the
	// fallback (strong) model was used instead.
	RoutedFallback RoutingDecision = "fallback"
)

// RoutingAttempt captures telemetry for a single model invocation.
type RoutingAttempt struct {
	Model        string           `json:"model"`
	Confidence   ConfidenceStatus `json:"confidence"`
	LatencyMS    float64          `json:"latency_ms"`
	InputTokens  int              `json:"input_tokens"`
	OutputTokens int              `json:"output_tokens"`
}

// RoutingResult is the full outcome of a RoutedParser.Parse call.
type RoutingResult struct {
	Decision    RoutingDecision  `json:"decision"`
	Primary     RoutingAttempt   `json:"primary"`
	Fallback    *RoutingAttempt  `json:"fallback,omitempty"`
	RulesJSON   string           `json:"rules_json"`
	Explanation string           `json:"explanation,omitempty"`
}

// RoutedParser runs the cheap primary model first and escalates to the strong
// fallback model when primary confidence is not OK.
type RoutedParser struct {
	primary       ChatClient
	fallback      ChatClient
	primaryModel  string
	fallbackModel string
}

// NewRoutedParser creates a RoutedParser.
// primaryModel and fallbackModel are human-readable labels used in results
// (they must match the model configured in the respective ChatClient).
func NewRoutedParser(primary, fallback ChatClient, primaryModel, fallbackModel string) *RoutedParser {
	return &RoutedParser{
		primary:       primary,
		fallback:      fallback,
		primaryModel:  primaryModel,
		fallbackModel: fallbackModel,
	}
}

// Parse runs the routing logic:
//  1. Call self_check with the primary client → get rulesJSON + SelfCheckResult.
//  2. Run constraint check on the rulesJSON → get ConstraintResult.
//  3. Compute overall confidence as the worst of the two.
//  4. If overall == OK → return RoutedPrimary.
//  5. Otherwise → repeat steps 1–3 with the fallback client → return RoutedFallback.
func (r *RoutedParser) Parse(ctx context.Context, phrase string) RoutingResult {
	primaryAttempt, rulesJSON, primaryOverall := r.runAttempt(ctx, r.primary, r.primaryModel, phrase)

	if primaryOverall == ConfidenceStatusOK {
		slog.InfoContext(ctx, "routing: primary model sufficient",
			slog.String("model", r.primaryModel),
			slog.String("confidence", string(primaryOverall)),
		)
		return RoutingResult{
			Decision:    RoutedPrimary,
			Primary:     primaryAttempt,
			RulesJSON:   rulesJSON,
			Explanation: primaryAttempt.Model + " confidence OK; no escalation needed",
		}
	}

	slog.InfoContext(ctx, "routing: escalating to fallback model",
		slog.String("primary_model", r.primaryModel),
		slog.String("primary_confidence", string(primaryOverall)),
		slog.String("fallback_model", r.fallbackModel),
	)

	fallbackAttempt, fallbackRulesJSON, _ := r.runAttempt(ctx, r.fallback, r.fallbackModel, phrase)

	slog.InfoContext(ctx, "routing: fallback complete",
		slog.String("model", r.fallbackModel),
		slog.String("confidence", string(fallbackAttempt.Confidence)),
	)

	return RoutingResult{
		Decision:    RoutedFallback,
		Primary:     primaryAttempt,
		Fallback:    &fallbackAttempt,
		RulesJSON:   fallbackRulesJSON,
		Explanation: "primary confidence " + string(primaryOverall) + "; escalated to " + r.fallbackModel,
	}
}

// runAttempt executes self_check + constraint for the given client and returns
// the RoutingAttempt, the best rulesJSON, and the overall ConfidenceStatus.
func (r *RoutedParser) runAttempt(
	ctx context.Context,
	client ChatClient,
	modelLabel string,
	phrase string,
) (RoutingAttempt, string, ConfidenceStatus) {
	start := time.Now()

	rulesJSON, scr := CheckSelfCheck(ctx, client, phrase)
	cr := CheckConstraint(ctx, rulesJSON)

	overall := worstStatus(scr.Status, cr.Status)
	latencyMS := float64(time.Since(start).Milliseconds())

	attempt := RoutingAttempt{
		Model:        modelLabel,
		Confidence:   overall,
		LatencyMS:    latencyMS,
		InputTokens:  scr.InputTokens,
		OutputTokens: scr.OutputTokens,
	}

	return attempt, rulesJSON, overall
}

// worstStatus returns the more severe of two ConfidenceStatus values.
// Severity order: FAIL > UNSURE > OK.
func worstStatus(a, b ConfidenceStatus) ConfidenceStatus {
	if a == ConfidenceStatusFail || b == ConfidenceStatusFail {
		return ConfidenceStatusFail
	}
	if a == ConfidenceStatusUnsure || b == ConfidenceStatusUnsure {
		return ConfidenceStatusUnsure
	}
	return ConfidenceStatusOK
}
