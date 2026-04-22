# Spec: Model Routing with Confidence-Based Fallback

## Goal
Day 8: Route LLM requests between cheap and strong models based on confidence from Day 7.

## Strategy (Variant B — both OpenAI)
1. **Primary** (cheap): `gpt-4o-mini` — parse phrase → check confidence (self_check + constraint)
2. **If confidence OK** → return result (stayed on cheap model)
3. **If confidence UNSURE or FAIL** → escalate to **fallback**: `gpt-4o` — re-parse + re-check
4. Return result with routing metadata

## Implementation

### Create: `services/api-dashboard/internal/llm/router.go`

```go
// RoutingDecision indicates which model handled the request.
type RoutingDecision string
const (
    RoutedPrimary  RoutingDecision = "primary"
    RoutedFallback RoutingDecision = "fallback"
)

// RoutingAttempt holds metrics for one model attempt.
type RoutingAttempt struct {
    Model       string
    Confidence  ConfidenceStatus
    LatencyMS   float64
    InputTokens int
    OutputTokens int
}

// RoutingResult is the full outcome of a routed parse.
type RoutingResult struct {
    Decision    RoutingDecision
    Primary     RoutingAttempt
    Fallback    *RoutingAttempt // nil if not escalated
    Rules       []model.PublisherRule
    Report      // from service layer
}

// RoutedParser wraps two ChatClients — primary (cheap) and fallback (strong).
type RoutedParser struct {
    primary  ChatClient
    fallback ChatClient
    tracer   trace.Tracer
}

// Parse: primary first → check confidence → if not OK → fallback
func (r *RoutedParser) Parse(ctx, phrase, approaches) → RoutingResult
```

### Create: `services/api-dashboard/finetune/routing_benchmark_test.go`

- Load corpus.jsonl
- Create 2 OpenAI clients: `gpt-4o-mini` + `gpt-4o`
- Run each phrase through RoutedParser
- Print table: phrase | group | primary_conf | routed_to | final_conf | latency
- Summary: X stayed on mini, Y escalated to gpt-4o

### Nothing else changes — 0 modifications to existing code.

## Acceptance Criteria
- [ ] RoutedParser escalates on UNSURE/FAIL confidence
- [ ] Benchmark prints routing decision table
- [ ] Correct phrases stay on mini, noisy/edge escalate
