# Implementation Log: Micro-Model First Inference

**Stage**: Implement  
**Date**: 2026-04-24  
**Agent**: go-builder (inline)

## Files Written

### New files

| File | Lines | Purpose |
|------|-------|---------|
| `services/api-dashboard/internal/llm/embed.go` | 64 | `EmbedClient` interface + `openAIEmbedClient` impl using `client.Embeddings.New` |
| `services/api-dashboard/internal/llm/micro.go` | 127 | `Intent` enum, `MicroResult`, `MicroClassifier`, `EmbedMicro.Classify`, exported `MeanVector` + `Cosine` helpers |
| `services/api-dashboard/internal/llm/micro_test.go` | 185 | Table-driven unit tests: Cosine (7 cases + panic), MeanVector (4 cases + panic), EmbedMicro.Classify (5 cases + error propagation) |
| `services/api-dashboard/internal/llm/two_level.go` | 107 | `Route` enum, `TwoLevelResult`, `TwoLevelParser.Parse` orchestrating Level-1 → Level-2 → optional self-check |
| `services/api-dashboard/internal/llm/two_level_test.go` | 180 | Table-driven unit tests for all 5 routing paths + micro error propagation test |
| `services/api-dashboard/finetune/build_prototypes_test.go` | 78 | `TestBuildPrototypes` — offline prototype builder, skipped unless `OPENAI_API_KEY` + `BUILD_PROTOTYPES=1` |
| `services/api-dashboard/finetune/microfirst_benchmark_test.go` | 200 | `TestMicroFirstBenchmark` — end-to-end vs baseline, writes `report_microfirst.md` |

## Decisions Made

### embed.go

- `EmbedResponse.Vector` is `[]float32` (API returns `[]float64`; converted during unmarshal to keep micro.go arithmetic cheap).
- Empty text → `fmt.Errorf("embed input: text must not be empty")` — no import of `model.ErrInvalidInput` to keep the `llm` package self-contained.
- Mirrors `openai.go` pattern exactly: `openai.NewClient(option.WithAPIKey(...))`, then `client.Embeddings.New(ctx, params)`.

### micro.go

- `MeanVector`: panics on dimension mismatch (programming error; callers must ensure uniformity). Returns `nil` on empty slice. Documented in godoc.
- `Cosine`: panics on dimension mismatch; returns `0` when either vector is zero (undefined direction). Both behaviours documented.
- Top-1/top-2 scan is O(|prototypes|) — 3 classes means 3 comparisons, negligible.
- `Margin` is 0 when there is only one prototype class (no top-2 candidate). Handled by checking `top2Intent != ""`.

### two_level.go

- `Parse` returns `(TwoLevelResult, error)` instead of just `TwoLevelResult`. Rationale: micro failure is a genuine infrastructure error (network, auth), not a domain FAIL. Callers can distinguish transient from semantic failures.
- `MultiStageParser.client` is accessible from `two_level.go` because both are in the same `llm` package.
- Self-check for `RouteLLMWithCheck`: full `CheckSelfCheck` call (2 LLM calls: parse + verify). The parse call in self-check redundantly calls LLM again — spec says "run the existing SelfCheck approach on top of multistage result; just annotate". This is the simplest implementation; a future optimisation would be to pass the already-parsed rules JSON directly to the verify prompt, skipping the parse step.
- Early-fail path: `RulesJSON=""`, `Confidence=FAIL`, `LLM=nil`, `UsedLLM=false`.

### two_level_test.go

- Reuses `mockChatClient` from `self_check_test.go` (same package, no redeclaration needed).
- `llmCallCount` is asserted at the end of each test case via `callIndex`.
- LLM response slice is pre-loaded per test case; unexpected extra calls return an error response and fail the test.

### build_prototypes_test.go

- Uses `t.Context()` (Go 1.21+; project runs Go 1.26).
- Path resolution via `runtime.Caller(0)` — same pattern as existing benchmark files.
- `prototypesPath()` is defined here and reused by `loadPrototypes` in `microfirst_benchmark_test.go`.

### microfirst_benchmark_test.go

- Named `microRow` struct instead of anonymous struct — Go does not allow anonymous struct parameters in function signatures.
- `loadPrototypes` skips the test with a clear message pointing to `TestBuildPrototypes` if the file is missing.
- LLM call count for `RouteLLMWithCheck` adds +2 for self-check (parse + verify calls). Estimate only — actual varies if multistage returns 0 rules early.
- Report written to `finetune/confidence-check/results/report_microfirst.md`.

## Validation Results

```
go build ./...          PASS
go vet ./...            PASS (no issues)
go test ./internal/llm/ PASS — 78 tests
go test ./...           PASS — 541 tests across 11 packages
```

New tests added: 30 (Cosine: 8, MeanVector: 6, EmbedMicro.Classify: 7, TwoLevelParser: 9).

No existing tests broken.

## Not Done (out of scope)

- Config wiring of `TwoLevelParser` into HTTP handler (next session).
- Live benchmark run (requires `OPENAI_API_KEY`).
- Prototype file generation (requires `OPENAI_API_KEY` + `BUILD_PROTOTYPES=1`).
