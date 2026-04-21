# Test Report: LLM Rule Parser

Stage: Test | Agent: go-test-writer

## Uncovered Modules

| Module | File | Existing test? |
|--------|------|---------------|
| Constraint checker | `internal/llm/constraint.go` | No |
| Self-check | `internal/llm/self_check.go` | No |
| Rule parser service | `internal/service/rule_parser.go` | No |
| Rule parser handler | `internal/handler/rule_parser.go` | No |

## Tests Written

| File | Test functions | Cases |
|------|---------------|-------|
| `internal/llm/constraint_test.go` | `TestCheckConstraint_ValidRules`, `TestCheckConstraint_InvalidJSON`, `TestCheckConstraint_UnknownRuleType`, `TestCheckConstraint_BlocklistInvalidConfig`, `TestCheckConstraint_FrequencyCapInvalidConfig`, `TestCheckConstraint_GeoFilterInvalidConfig`, `TestCheckConstraint_PlatformFilterInvalidConfig`, `TestCheckConstraint_Latency` | 33 |
| `internal/llm/self_check_test.go` | `TestParseSelfCheckResponse`, `TestCheckSelfCheck_YesVerdict`, `TestCheckSelfCheck_NoVerdict`, `TestCheckSelfCheck_ParseCallFails`, `TestCheckSelfCheck_VerifyCallFails`, `TestCheckSelfCheck_UnparseableVerifyResponse`, `TestCheckSelfCheck_LatencyPopulated` | 15 |
| `internal/service/rule_parser_test.go` | `TestResolveApproaches_*` (5), `TestOverallVerdict`, `TestRuleParserService_Parse_*` (9) | 20 |
| `internal/handler/rule_parser_test.go` | `TestRuleParserHandler_Parse_*` (8) | 8 |

Total new cases: **76**

## Test Results

```
llm package:     33 passed
service package: 20 passed
handler package:  8 passed

Full suite: 511 passed in 11 packages — 0 failures
```

## Coverage Details

### `internal/llm/constraint.go`
- All 5 rule types tested with valid configs (blocklist, allowlist, frequency_cap, geo_filter, platform_filter)
- platform_filter: all platform values (ios, android, web, ctv) covered; unknown value "windows" tested
- geo_filter: both modes (include/exclude) valid; invalid mode, empty/missing country_codes
- frequency_cap: zero max_impressions, negative window_seconds
- Invalid JSON: malformed string, empty string, object instead of array, empty array, null
- Unknown rule type: produces FAIL with non-empty Errors slice
- Latency field is populated on both success and failure paths

### `internal/llm/self_check.go`
- `parseSelfCheckResponse`: 8 cases including YES/NO/lowercase/spaces/no-explanation/multiline/empty/unparseable
- `CheckSelfCheck`: YES verdict → status OK + token accumulation; NO verdict → UNSURE; parse LLM call fails → FAIL with empty rulesJSON; verify LLM call fails → UNSURE with "verify call failed" explanation; unparseable verify response → UNSURE; Latency populated

### `internal/service/rule_parser.go`
- `resolveApproaches`: nil input, all-invalid input, single constraint, single self_check, deduplication
- `overallVerdict`: all-OK, one-UNSURE, one-FAIL-dominates, empty map
- `Parse`: empty phrase → ErrInvalidInput, whitespace-only → ErrInvalidInput, constraint-only valid → OK, constraint-only invalid config → FAIL, self_check YES → OK, self_check NO → UNSURE, both approaches default (nil) → two approaches in report, LLM call failure in constraint-only path, multiple rules parsed correctly, invalid approach names fall back to defaults

### `internal/handler/rule_parser.go`
- `NewRuleParserHandlerDisabled()` → 501 NOT_CONFIGURED
- Missing phrase field → 400 INVALID_INPUT
- Empty phrase string → 400 INVALID_INPUT
- Malformed request body → 400 INVALID_BODY
- Valid request with explicit approaches → 200 with rules + confidence
- Valid request with no approaches → 200, both defaults in confidence report
- Content-Type header set to application/json on success
- Frequency cap rule round-trip

## Bugs Discovered

None.
