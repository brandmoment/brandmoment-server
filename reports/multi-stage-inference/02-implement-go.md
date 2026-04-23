# Implementation: Multi-Stage Inference

## Status: Done

## Files Created

### 1. `services/api-dashboard/internal/llm/multistage.go`
Package `llm`. Implements the 3-stage pipeline.

**Types exported:**
- `StageResult` — per-stage metrics (stage name, latency ms, input/output tokens)
- `MultiStageResult` — full outcome (rules JSON, stages slice, confidence status, totals)
- `MultiStageParser` — holds a `ChatClient`

**Functions exported:**
- `NewMultiStageParser(client ChatClient) *MultiStageParser`
- `(*MultiStageParser).Parse(ctx, phrase) MultiStageResult`

**Internal helpers:**
- `analyzePhrase(ctx, phrase)` — Stage 1 LLM call, returns `analysisResult` + `StageResult`
- `extractConfig(ctx, phrase, ruleType, summary)` — Stage 2 LLM call per rule, returns `json.RawMessage` + `StageResult`
- `assembleRules(analyses, configs)` — Stage 3 pure Go, calls `CheckConstraint`

**Stage 1 system prompt** lists only the 5 valid type names, requests `{"count":N,"rules":[{"type":"...","summary":"..."},...]}`. Temperature 0.

**Stage 2 system prompt** is parameterised by rule type. Shows only the config schema for that type. User message combines original phrase + stage-1 summary. Temperature 0.

**Stage 3** marshals `[]{"type","config"}` and passes through `CheckConstraint` to produce `ConfidenceStatus`.

All LLM responses stripped via `StripMarkdownFences`. Each stage logged with `slog.InfoContext`.

### 2. `services/api-dashboard/finetune/multistage_benchmark_test.go`
Package `finetune`. Test function `TestMultiStageBenchmark`.

- Requires `OPENAI_API_KEY`, skips if absent.
- Creates `llm.NewOpenAIClient(key, "gpt-4o-mini")` — same model for fair comparison.
- Loads corpus via existing `loadCorpus(t)`.
- For each phrase runs both approaches sequentially:
  - Monolithic: `client.Complete` + `llm.CheckConstraint`
  - Multi-stage: `llm.NewMultiStageParser(client).Parse`
- Counts OK/UNSURE/FAIL per approach, accumulates latency and tokens.
- Prints per-phrase table + summary to stdout.
- Writes `finetune/confidence-check/results/report_multistage.md`.

## Validation

```
go build ./internal/llm/...   → Success
go build ./...                → Success
go vet ./internal/llm/... ./finetune/... → No issues
```

## Next Steps

```
OPENAI_API_KEY=sk-... go test -v -run TestMultiStageBenchmark ./finetune/
```
