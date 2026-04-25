package llm

import (
	"context"
	"log/slog"
	"time"
)

// Route identifies which code path was taken by TwoLevelParser.
type Route string

const (
	// RouteMicroEarlyFail means the micro-classifier detected an invalid phrase with
	// sufficient margin and the LLM was not called.
	RouteMicroEarlyFail Route = "micro_early_fail"
	// RouteLLMDirect is kept for backward compatibility but is no longer produced
	// by this version of the router (dead code — left for minimal diff).
	RouteLLMDirect Route = "llm_direct"
	// RouteLLMWithCheck means the phrase was ambiguous or low-margin, so the LLM
	// was called and a self-check was run on the result.
	RouteLLMWithCheck Route = "llm_with_check"
	// RouteMicroAnswer means the micro-classifier was confident about the rule type,
	// so only ExtractOnly (1 LLM call) was used — no analyze stage, no self-check.
	RouteMicroAnswer Route = "micro_answer"
)

// TwoLevelResult is the outcome of TwoLevelParser.Parse.
type TwoLevelResult struct {
	// RulesJSON is the parsed rule array; empty string on early-fail.
	RulesJSON string
	// Confidence is the final confidence status.
	Confidence ConfidenceStatus
	// Route describes which decision path was taken.
	Route Route
	// Micro contains the Level-1 classifier result.
	Micro MicroResult
	// LLM contains the Level-2 multi-stage parse result; nil when early-exit.
	LLM *MultiStageResult
	// TotalLatency is wall-clock time in milliseconds for the whole Parse call.
	TotalLatency float64
	// UsedLLM is true when the LLM was invoked.
	UsedLLM bool
}

// TwoLevelParser orchestrates the two-level inference pipeline:
//  1. MicroClassifier — cheap intent gate (embedding-based).
//  2. MultiStageParser — full LLM parse (only when needed).
type TwoLevelParser struct {
	micro         MicroClassifier
	llm           *MultiStageParser
	marginFloor   float64
	ruleTypeFloor float64
}

// NewTwoLevelParser creates a TwoLevelParser.
// marginFloor is the minimum cosine-margin to trust an invalid/ambiguous gate decision.
// ruleTypeFloor is the higher threshold required to trust a specific rule-type prediction
// and take the RouteMicroAnswer path (extract-only, 1 LLM call, no self-check).
func NewTwoLevelParser(micro MicroClassifier, llm *MultiStageParser, marginFloor, ruleTypeFloor float64) *TwoLevelParser {
	return &TwoLevelParser{
		micro:         micro,
		llm:           llm,
		marginFloor:   marginFloor,
		ruleTypeFloor: ruleTypeFloor,
	}
}

// Parse runs the two-level pipeline. It never returns an error; failures are
// encoded as ConfidenceStatusFail in the result.
func (p *TwoLevelParser) Parse(ctx context.Context, phrase string) (TwoLevelResult, error) {
	start := time.Now()

	// Level 1: micro classify.
	micro, err := p.micro.Classify(ctx, phrase)
	if err != nil {
		return TwoLevelResult{}, err
	}

	slog.InfoContext(ctx, "two_level micro done",
		slog.String("intent", string(micro.Intent)),
		slog.Float64("top1", micro.Top1),
		slog.Float64("margin", micro.Margin),
	)

	// Routing logic.
	//
	// Table:
	//   invalid  + margin≥marginFloor → RouteMicroEarlyFail (no LLM)
	//   invalid  + margin<marginFloor → RouteLLMWithCheck
	//   ambiguous (any margin)        → RouteLLMWithCheck
	//   {rule-type} + margin≥ruleTypeFloor → RouteMicroAnswer (ExtractOnly, 1 LLM call)
	//   {rule-type} + margin<ruleTypeFloor → RouteLLMWithCheck

	if micro.Intent == IntentInvalid && micro.Margin >= p.marginFloor {
		// Early FAIL: micro is confident this is invalid — do not call LLM.
		totalMS := float64(time.Since(start).Milliseconds())
		slog.InfoContext(ctx, "two_level early fail",
			slog.Float64("margin", micro.Margin),
			slog.Float64("total_latency_ms", totalMS),
		)
		return TwoLevelResult{
			RulesJSON:    "",
			Confidence:   ConfidenceStatusFail,
			Route:        RouteMicroEarlyFail,
			Micro:        micro,
			LLM:          nil,
			TotalLatency: totalMS,
			UsedLLM:      false,
		}, nil
	}

	// Check whether this is a rule-type intent (not invalid/ambiguous).
	isRuleType := micro.Intent == IntentBlocklist ||
		micro.Intent == IntentAllowlist ||
		micro.Intent == IntentGeoFilter ||
		micro.Intent == IntentPlatformFilter ||
		micro.Intent == IntentFrequencyCap

	if isRuleType && micro.Margin >= p.ruleTypeFloor {
		// MicroAnswer: micro is confident about the rule type — skip analyze,
		// call ExtractOnly (1 LLM call), no self-check.
		llmResult := p.llm.ExtractOnly(ctx, phrase, string(micro.Intent))

		slog.InfoContext(ctx, "two_level micro_answer done",
			slog.String("intent", string(micro.Intent)),
			slog.String("confidence", string(llmResult.Confidence)),
		)

		totalMS := float64(time.Since(start).Milliseconds())
		return TwoLevelResult{
			RulesJSON:    llmResult.RulesJSON,
			Confidence:   llmResult.Confidence,
			Route:        RouteMicroAnswer,
			Micro:        micro,
			LLM:          &llmResult,
			TotalLatency: totalMS,
			UsedLLM:      true,
		}, nil
	}

	// All remaining cases → full multi-stage LLM parse with self-check.
	llmResult := p.llm.Parse(ctx, phrase)

	slog.InfoContext(ctx, "two_level llm done",
		slog.String("route", string(RouteLLMWithCheck)),
		slog.String("confidence", string(llmResult.Confidence)),
	)

	finalConfidence := llmResult.Confidence

	// Run a lightweight self-check: if it disagrees (NO), downgrade OK → UNSURE.
	if llmResult.Confidence == ConfidenceStatusOK && llmResult.RulesJSON != "" {
		_, scr := CheckSelfCheck(ctx, p.llm.client, phrase)
		slog.InfoContext(ctx, "two_level self_check done",
			slog.String("self_check_status", string(scr.Status)),
		)
		if scr.Status != ConfidenceStatusOK {
			finalConfidence = ConfidenceStatusUnsure
		}
	}

	totalMS := float64(time.Since(start).Milliseconds())
	return TwoLevelResult{
		RulesJSON:    llmResult.RulesJSON,
		Confidence:   finalConfidence,
		Route:        RouteLLMWithCheck,
		Micro:        micro,
		LLM:          &llmResult,
		TotalLatency: totalMS,
		UsedLLM:      true,
	}, nil
}
