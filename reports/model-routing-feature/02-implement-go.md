# Stage: Implement — go-builder

## Result: SUCCESS

`go build ./...` — OK  
`go vet ./...` — OK (0 issues)

---

## Files Created

### 1. `services/api-dashboard/internal/llm/router.go`

Package `llm`. Exports:

- `RoutingDecision` string type with constants `RoutedPrimary` / `RoutedFallback`
- `RoutingAttempt` struct — model label, confidence, latency_ms, token counts
- `RoutingResult` struct — decision, primary attempt, optional fallback attempt, rules_json, explanation
- `RoutedParser` struct — holds two `ChatClient` values (primary + fallback) with model labels
- `NewRoutedParser(primary, fallback ChatClient, primaryModel, fallbackModel string) *RoutedParser`
- `(*RoutedParser).Parse(ctx, phrase) RoutingResult` — runs self_check + constraint on primary; escalates to fallback if overall confidence != OK
- `worstStatus(a, b ConfidenceStatus) ConfidenceStatus` — internal helper, FAIL > UNSURE > OK
- `runAttempt(ctx, client, label, phrase)` — internal helper, calls CheckSelfCheck + CheckConstraint, measures latency

Logging: `slog.InfoContext` at routing decision points (primary sufficient / escalating / fallback complete).

### 2. `services/api-dashboard/finetune/routing_benchmark_test.go`

Package `finetune`. Exports (test-only):

- `routingRow` struct — package-level, used by both `TestRoutingBenchmark` and `writeRoutingReportMD`
- `TestRoutingBenchmark(t)` — skips if `OPENAI_API_KEY` unset; creates gpt-4o-mini primary + gpt-4o fallback; loads corpus via existing `loadCorpus`; runs `router.Parse` per entry; prints ASCII table + summary; calls `writeRoutingReportMD`
- `writeRoutingReportMD(t, rows, ...)` — writes `finetune/confidence-check/results/report_routing.md`

---

## Key Design Decisions

- `runAttempt` is a private method so `Parse` is clean and the logic is not duplicated between primary and fallback paths.
- Token counts in `RoutingResult.Primary` reflect only self_check calls (constraint adds no tokens); this matches the existing pattern in `SelfCheckResult`.
- `worstStatus` mirrors `overallVerdict` in `service/rule_parser.go` but without the `map` iteration overhead, since we always have exactly two statuses to compare.
- The `routingRow` type is hoisted to package scope to avoid the anonymous-struct mismatch that caused the initial vet failure.

---

## Next Steps

1. `go test -v -run TestRoutingBenchmark ./finetune/` (requires `OPENAI_API_KEY`)
2. Review `finetune/confidence-check/results/report_routing.md` after run
