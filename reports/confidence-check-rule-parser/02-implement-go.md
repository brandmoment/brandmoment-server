# Implementation Report: Confidence-Check Rule Parser

**Date**: 2026-04-21  
**Branch**: `feature/confidence-check-rule-parser`  
**Status**: Complete — `go build` + `go vet` + `go test` all green (435 tests passed)

---

## What Was Built

### New packages / files

| File | Purpose |
|---|---|
| `internal/llm/client.go` | `ChatClient` interface, `ChatRequest`/`ChatResponse` types, `Provider`/`Role`/`ApproachName`/`ConfidenceStatus` enums |
| `internal/llm/prompt.go` | Shared `SystemPrompt` constant describing the 5 rule types |
| `internal/llm/confidence.go` | `ConfidenceStatus` and `ApproachName` constants |
| `internal/llm/openai.go` | OpenAI implementation (`github.com/openai/openai-go`) |
| `internal/llm/gemini.go` | Gemini implementation (`github.com/google/generative-ai-go/genai`) |
| `internal/llm/constraint.go` | Constraint approach: `json.Unmarshal` + structural rule validation, no LLM call |
| `internal/llm/scoring.go` | Scoring approach: model returns `confidence: 0.0–1.0` + `reasoning` |
| `internal/llm/self_check.go` | Self-check approach: 2 calls — parse then verify |
| `internal/llm/redundancy.go` | Redundancy approach: 3 parallel calls at temperature=0.7, majority vote |
| `internal/service/rule_parser.go` | `RuleParserService.Parse(ctx, phrase, approaches)` orchestration |
| `internal/handler/rule_parser.go` | HTTP handler for `POST /v1/publisher-rules/parse`, 501 if key absent |
| `finetune/confidence_benchmark_test.go` | Go test harness — runs all approaches × provider, prints table, writes MD report |
| `../../finetune/confidence-check/data/corpus.jsonl` | 30 phrases: 15 correct, 10 edge, 5 noisy |

### Modified existing files

| File | Change |
|---|---|
| `internal/config/config.go` | Added `OpenAIAPIKey`, `GeminiAPIKey` fields (from `OPENAI_API_KEY`, `GEMINI_API_KEY` env) |
| `internal/router/router.go` | Added `RuleParser *handler.RuleParserHandler` to `Handlers`; registered `POST /v1/publisher-rules/parse` (editor+) |
| `cmd/server/main.go` | LLM client + `RuleParserService` + handler wired into DI; prefers OpenAI key, falls back to Gemini, falls back to disabled (501) |
| `go.mod` / `go.sum` | Added `github.com/openai/openai-go` and `github.com/google/generative-ai-go` |

---

## Architecture Decisions

**Constraint approach** lives in `internal/llm/constraint.go` rather than importing `service.validateRuleConfig`. Reason: `service` is not importable from `llm` (cycle risk). The validation logic is a verbatim mirror — simple enough that duplication is correct here.

**Rules JSON sharing**: the service runs approaches in the requested order. The first approach that produces a JSON string sets `rulesJSON`; subsequent approaches (including constraint) reuse it. This avoids redundant LLM calls.

**Disabled handler**: when neither API key is set, `NewRuleParserHandlerDisabled()` creates a handler with `service == nil`. The `Parse` method checks and returns 501 immediately.

**Benchmark location**: `services/api-dashboard/finetune/` (inside the module) so it can import `internal/llm` and `internal/service`. The corpus data lives at `finetune/confidence-check/data/corpus.jsonl` (repo root) and is located via `runtime.Caller(0)` path resolution.

---

## HTTP API

```
POST /v1/publisher-rules/parse
Authorization: Bearer <jwt>
X-Org-ID: <uuid>
Content-Type: application/json

{
  "phrase": "Block gambling in Russia max 3 times per day",
  "approaches": ["constraint", "scoring", "self_check", "redundancy"]
}
```

Response `200`:
```json
{
  "data": {
    "rules": [
      {"type": "blocklist", "config": {"categories": ["gambling"]}},
      {"type": "geo_filter", "config": {"mode": "exclude", "country_codes": ["RU"]}},
      {"type": "frequency_cap", "config": {"max_impressions": 3, "window_seconds": 86400}}
    ],
    "confidence": {
      "overall": "OK",
      "approaches": {
        "scoring":    {"status": "OK",  "latency_ms": "1240.0", "input_tokens": 312, "output_tokens": 89, "detail": {"confidence": 0.92, "reasoning": "..."}},
        "self_check": {"status": "OK",  "latency_ms": "2100.0", "input_tokens": 480, "output_tokens": 34, "detail": {"explanation": "YES ..."}},
        "redundancy": {"status": "OK",  "latency_ms": "1890.0", "input_tokens": 936, "output_tokens": 267, "detail": {"matches": 3, "runs": 3}},
        "constraint": {"status": "OK",  "latency_ms": "0.1",    "input_tokens": 0,   "output_tokens": 0,  "detail": {"errors": null}}
      }
    }
  }
}
```

Response `501` when no API key is configured:
```json
{"error": {"code": "NOT_CONFIGURED", "message": "LLM API key not configured"}}
```

---

## Running the Benchmark

```bash
# OpenAI
OPENAI_API_KEY=sk-... go test -v -run TestBenchmark \
  ./services/api-dashboard/finetune/

# Gemini
GEMINI_API_KEY=... go test -v -run TestBenchmark \
  ./services/api-dashboard/finetune/
```

Output is printed to stdout and also saved to:
`finetune/confidence-check/results/report_<provider>.md`

---

## Validation Results

```
go build ./...   — OK
go vet ./...     — OK (no issues)
go test ./...    — 435 passed, 11 packages
```

No existing tests were broken. The finetune test skips automatically when no API key is present.

---

## Next Steps

1. Run live benchmark with real keys → compare approaches in `results/report_*.md`
2. Add `go-test-writer` unit tests for `llm/scoring.go`, `llm/self_check.go`, `llm/redundancy.go`
3. Build `apps/dashboard/app/(dashboard)/publisher-rules/parse/page.tsx` UI
4. Consider caching: redundancy makes 3 calls per phrase; could cache by phrase hash
