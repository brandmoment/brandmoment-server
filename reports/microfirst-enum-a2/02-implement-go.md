# A2 Implementation Report

## Files Changed

| File | Change |
|------|--------|
| `services/api-dashboard/internal/llm/micro.go` | Replaced 3-class Intent enum (valid/ambiguous/gibberish) with 7-class A2 taxonomy (blocklist/allowlist/geo_filter/platform_filter/frequency_cap/ambiguous/invalid) |
| `services/api-dashboard/internal/llm/two_level.go` | Added `RouteMicroAnswer` constant; added `ruleTypeFloor float64` field; extended `NewTwoLevelParser` to 4-arg signature; rewrote routing to implement spec table (invalid+high-margin→early-fail, ambiguous→llm_with_check, rule-type+high-margin→micro_answer, else→llm_with_check) |
| `services/api-dashboard/internal/llm/multistage.go` | Added `ExtractOnly` method — skips analyze stage, calls `extractConfig` once with caller-supplied ruleType, calls `assembleRules`, returns TotalCalls=1 |
| `services/api-dashboard/internal/llm/micro_test.go` | Updated `testPrototypes` from 3-class to 7-class orthogonal unit vectors; updated all test cases to use new Intent constants; added `TestEmbedMicro_AllSevenIntents` table-driven test |
| `services/api-dashboard/internal/llm/two_level_test.go` | Updated all test cases to use new Intent constants (IntentInvalid/IntentBlocklist); added `wantLLMTotalCalls` field; added cases for invalid+high-margin→RouteMicroEarlyFail, ambiguous→RouteLLMWithCheck, blocklist+high-margin→RouteMicroAnswer (TotalCalls==1), blocklist+low-margin→RouteLLMWithCheck; updated constructor call to 4-arg form |
| `services/api-dashboard/finetune/build_prototypes_test.go` | Changed `groupToIntent` map to per-entry label derivation: `correct→expected[0].type`, `edge→IntentAmbiguous`, `noisy→IntentInvalid` — produces 7-entry prototypes.json |
| `services/api-dashboard/finetune/microfirst_benchmark_test.go` | Added `RULE_TYPE_FLOOR` env parsing (default 0.05); updated `NewTwoLevelParser` call to 4-arg; added `RouteMicroAnswer` to route table in console summary and report; added "% without full pipeline" summary metric; updated report filename to `report_microfirst_a2_margin{floor}_rtf{rtf}.md` |

## Validation Output

```
go build ./services/api-dashboard/...
# BUILD_OK (no errors)

go vet ./services/api-dashboard/...
# VET_OK (no issues)

go test ./services/api-dashboard/internal/llm/...
# 87 passed, 0 failed

go test ./services/api-dashboard/... -short
# 550 passed in 11 packages, 0 failed
```

## Deviations from Spec

**None.** All spec requirements implemented verbatim:

- Intent string values match `expected[].type` verbatim (`"blocklist"`, `"allowlist"`, `"geo_filter"`, `"platform_filter"`, `"frequency_cap"`, `"ambiguous"`, `"invalid"`).
- `RouteLLMDirect` kept as dead-code constant for backward-compat (spec explicitly allows this).
- `ExtractOnly` stage name uses `extract_only:<type>` prefix per spec.
- `TotalCalls = 1` for ExtractOnly (assemble is counted in Stages but not in TotalCalls, matching spec).
- `ruleTypeFloor` default = 0.05, `marginFloor` default = 0.02, both env-tunable.

## Next Steps for Main

1. **Rebuild prototypes** (requires `OPENAI_API_KEY`):
   ```
   BUILD_PROTOTYPES=1 OPENAI_API_KEY=sk-... go test -v -run TestBuildPrototypes ./services/api-dashboard/finetune/
   ```
   This will produce `finetune/confidence-check/data/prototypes.json` with 7 class entries.

2. **Run A2 benchmark** (requires `OPENAI_API_KEY`):
   ```
   OPENAI_API_KEY=sk-... MARGIN_FLOOR=0.02 RULE_TYPE_FLOOR=0.05 go test -v -run TestMicroFirstBenchmark ./services/api-dashboard/finetune/
   ```
   Report saved to `finetune/confidence-check/results/report_microfirst_a2_margin0020_rtf0050.md`.

3. **Tune `RULE_TYPE_FLOOR`** upward (0.08–0.10) if benchmark shows allowlist misclassifications on the micro_answer path.

4. **Compare** vs Day 9 monolith and Day 10 floor=0.02 results → write `03-report.md`.
