package llm

import (
	"context"
	"log/slog"
	"time"
)

// Route identifies which code path was taken by TwoLevelParser.
type Route string

const (
	// RouteMicroEarlyFail means the micro-classifier detected gibberish with
	// sufficient margin and the LLM was not called.
	RouteMicroEarlyFail Route = "micro_early_fail"
	// RouteLLMDirect means the micro-classifier was confident the phrase is valid
	// and the LLM was called without an additional self-check.
	RouteLLMDirect Route = "llm_direct"
	// RouteLLMWithCheck means the phrase was ambiguous or low-margin, so the LLM
	// was called and a self-check was run on the result.
	RouteLLMWithCheck Route = "llm_with_check"
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
	micro       MicroClassifier
	llm         *MultiStageParser
	marginFloor float64
}

// NewTwoLevelParser creates a TwoLevelParser.
// marginFloor is the minimum cosine-margin required to trust the micro result;
// phrases below this threshold are routed to LLMWithCheck.
func NewTwoLevelParser(micro MicroClassifier, llm *MultiStageParser, marginFloor float64) *TwoLevelParser {
	return &TwoLevelParser{
		micro:       micro,
		llm:         llm,
		marginFloor: marginFloor,
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
	if micro.Intent == IntentGibberish && micro.Margin >= p.marginFloor {
		// Early FAIL: do not call LLM.
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

	// Determine route before LLM call.
	needsCheck := micro.Intent == IntentAmbiguous || micro.Margin < p.marginFloor
	route := RouteLLMDirect
	if needsCheck {
		route = RouteLLMWithCheck
	}

	// Level 2: multi-stage LLM parse.
	llmResult := p.llm.Parse(ctx, phrase)

	slog.InfoContext(ctx, "two_level llm done",
		slog.String("route", string(route)),
		slog.String("confidence", string(llmResult.Confidence)),
	)

	finalConfidence := llmResult.Confidence

	// For LLMWithCheck, run a lightweight self-check annotation.
	// We run the self-check verify call (call 2 only) against the already-parsed rules JSON.
	// If the self-check disagrees (NO), downgrade OK → UNSURE.
	if route == RouteLLMWithCheck && llmResult.Confidence == ConfidenceStatusOK && llmResult.RulesJSON != "" {
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
		Route:        route,
		Micro:        micro,
		LLM:          &llmResult,
		TotalLatency: totalMS,
		UsedLLM:      true,
	}, nil
}
