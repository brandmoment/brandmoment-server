# Validation Report — A2 Micro-First Enum Classifier

Date: 2026-04-24
Stage: Validate
Agent: test-runner

## Commands Run

| # | Command | Exit | Duration |
|---|---------|------|----------|
| 1 | `go build github.com/brandmoment/brandmoment-server/services/api-dashboard/... github.com/brandmoment/brandmoment-server/packages/shared-domain/...` | 0 | ~1s |
| 2 | `go vet github.com/brandmoment/brandmoment-server/services/api-dashboard/... github.com/brandmoment/brandmoment-server/packages/shared-domain/...` | 0 | ~1s |
| 3 | `go test github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/llm/...` | 0 | 0.391s |

Note: `./...` pattern is not usable from workspace root with go.work (expected — pattern must reference module paths explicitly).

## Results

| Check | Status | Details |
|-------|--------|---------|
| go build | PASS | No errors across both workspace modules |
| go vet | PASS | No issues |
| go test ./internal/llm/ | PASS | All tests passed |

## Test Summary

Package: `github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/llm`
Result: `ok` — 0.391s

Tests passed:
- `TestCheckConstraint_*` — 16 subtests (valid rules, invalid JSON, unknown type, invalid configs)
- `TestCosine` — 7 subtests including zero-vector and panic guard
- `TestMeanVector` — 4 subtests
- `TestEmbedMicro_Classify` — 6 subtests covering 4 new intents + error path
- `TestEmbedMicro_AllSevenIntents` — 7 subtests: blocklist, allowlist, geo_filter, platform_filter, frequency_cap, ambiguous, invalid
- `TestParseSelfCheckResponse` — 8 subtests
- `TestCheckSelfCheck_*` — 6 tests
- `TestTwoLevelParser_Parse` — 6 subtests: invalid+high-margin→early-fail, invalid+low-margin→llm_with_check, ambiguous+high-margin→llm_with_check, blocklist+high-margin→micro_answer (TotalCalls=1), blocklist+low-margin→llm_with_check, error propagation
- `TestTwoLevelParser_MicroError_Propagated`

Total: 0 failed.

## Failures

None.

## Routing Decision

All three checks green → Validate → Report (no rework required).
