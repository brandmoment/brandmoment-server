# Spec: Micro-First Enum Classifier (A2)

## Motivation

Day 10 v1 (`reports/microfirst-inference/`) misaligns with LLM-course task
spec. Task required: "micro decides → LLM only when UNSURE/bad format". Our
v1 implementation routes valid-group phrases through full multi-stage LLM
anyway, because Intent ∈ {valid, ambiguous, gibberish} doesn't carry enough
information to produce the JSON output.

v2 (A2) moves the decision into micro by expanding the intent taxonomy to
match the rule-type enum. For high-confidence predictions, micro's answer
becomes the rule type directly, and only 1 LLM call (extract config) remains.

## Changes

### 1. Intent taxonomy (3 → 7)

```go
IntentBlocklist       Intent = "blocklist"
IntentAllowlist       Intent = "allowlist"
IntentGeoFilter       Intent = "geo_filter"
IntentPlatformFilter  Intent = "platform_filter"
IntentFrequencyCap    Intent = "frequency_cap"
IntentAmbiguous       Intent = "ambiguous"
IntentInvalid         Intent = "invalid"   // renamed from gibberish
```

Remove `IntentValid` and `IntentGibberish` (supersede).

### 2. Prototype labeling (corpus → prototypes)

| Group | Intent label source |
|-------|---------------------|
| correct (15) | `expected[0].type` verbatim (covers 5 rule types) |
| edge (10) | `ambiguous` (always — multi-rule or unparseable) |
| noisy (5) | `invalid` |

Per-class phrase counts after split:
- blocklist: 4 (rows 1, 6, 10, 14)
- geo_filter: 3 (rows 4, 5, 11)
- platform_filter: 3 (rows 2, 9, 13)
- frequency_cap: 3 (rows 3, 8, 12)
- allowlist: 2 (rows 7, 15)  ← thinnest
- ambiguous: 10
- invalid: 5

allowlist is thin; accept risk. If benchmark shows allowlist misclassified,
follow-up: add 1-2 synthetic allowlist phrases.

### 3. MultiStageParser.ExtractOnly (new method)

```go
func (p *MultiStageParser) ExtractOnly(ctx context.Context, phrase, ruleType string) MultiStageResult
```

- Skips stage 1 (analyze).
- Calls stage 2 (`extractConfig`) once with the known `ruleType`.
- Calls stage 3 (`assembleRules`) for single-rule JSON + constraint check.
- Returns MultiStageResult with `TotalCalls = 1`.

### 4. TwoLevelParser routing (new logic)

New field: `ruleTypeFloor float64` — separate (higher) threshold for trusting
a specific rule-type prediction vs the coarser invalid/ambiguous gate.

Routing table:

| Micro Intent | Margin condition | Route | LLM calls |
|--------------|------------------|-------|-----------|
| invalid | ≥ marginFloor | `micro_early_fail` | 0 |
| invalid | < marginFloor | `llm_with_check` | 2-4 + self_check |
| ambiguous | any | `llm_with_check` | 2-4 + self_check |
| {rule-type} | ≥ ruleTypeFloor | **`micro_answer`** (NEW) | **1 (extract-only)** |
| {rule-type} | < ruleTypeFloor | `llm_with_check` | 2-4 + self_check |

Constructor signature change:
```go
NewTwoLevelParser(micro, llm, marginFloor, ruleTypeFloor float64)
```

Default `ruleTypeFloor = 0.05` (higher than `marginFloor=0.02` to demand
more confidence for the decisive path). Tunable via `RULE_TYPE_FLOOR` env.

### 5. New Route constant

```go
RouteMicroAnswer Route = "micro_answer"
```

### 6. Benchmark metric

Add to report: **% handled without full pipeline** = (micro_early_fail +
micro_answer) / total. Target ≥ 50% for spec compliance.

## Acceptance Criteria

1. `go build ./... && go vet ./... && go test ./...` all green.
2. Unit tests updated for 7 Intents + new route.
3. Prototype builder produces 7-entry `prototypes.json`.
4. Benchmark runner emits per-phrase table including `micro_answer` rows.
5. New report saved as `report_microfirst_a2.md`.
6. Comparison vs Day 9 monolith + Day 10 floor=0.02 in `03-report.md`.

## Out of scope

- Changing stage 1 (analyze) prompt. A2 only skips it when micro is confident.
- Expanding corpus beyond 30 phrases.
- Multi-rule phrases — all multi-rule (edge group) routes through
  llm_with_check regardless.
- Wiring TwoLevelParser into HTTP handler.

## Risks

- **Allowlist thin (2 phrases)**: prototype may be a weak cluster center →
  misclassifications. Mitigation: observe in benchmark; add synthetic
  phrases if accuracy drops.
- **Rule-type confusion**: `Allow only iOS` (platform_filter) vs `Allow only
  mobile` (edge/ambiguous) — both use "Allow only". Micro might confuse.
  Mitigation: ruleTypeFloor=0.05 demands margin; low-margin → llm_with_check.
- **Accuracy regression**: if micro_answer chooses wrong type,
  extract-only produces invalid config. No recovery path (no analyze
  re-check). Mitigation: tune ruleTypeFloor upward (0.08-0.10) if regressions
  appear.

## Files touched

- `services/api-dashboard/internal/llm/micro.go` (Intent enum)
- `services/api-dashboard/internal/llm/two_level.go` (routing + new route)
- `services/api-dashboard/internal/llm/multistage.go` (new ExtractOnly)
- `services/api-dashboard/internal/llm/micro_test.go` (update test cases)
- `services/api-dashboard/internal/llm/two_level_test.go` (update routing cases)
- `services/api-dashboard/finetune/build_prototypes_test.go` (7-class labels)
- `services/api-dashboard/finetune/microfirst_benchmark_test.go` (new route + metric)
