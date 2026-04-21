# Cleanup: Remove scoring and redundancy confidence approaches

## Summary

Removed the `scoring` and `redundancy` confidence approaches from the codebase. Only `constraint` and `self_check` remain.

## Files deleted

- `services/api-dashboard/internal/llm/scoring.go` — `CheckScoring`, `ScoringResult`, `splitScoringResponse`, prompt constant
- `services/api-dashboard/internal/llm/redundancy.go` — `CheckRedundancy`, `RedundancyResult`, `normalizeJSON`, `sortedJSON`

## Files modified

### `services/api-dashboard/internal/llm/confidence.go`
- Removed `ApproachScoring ApproachName = "scoring"` constant
- Removed `ApproachRedundancy ApproachName = "redundancy"` constant

### `services/api-dashboard/internal/service/rule_parser.go`
- Removed `ApproachScoring` case from the approach switch (including `llm.CheckScoring` call and its report entry)
- Removed `ApproachRedundancy` case from the approach switch (including `llm.CheckRedundancy` call and its report entry)
- Updated `resolveApproaches`: defaults now `[self_check, constraint]` (was all four)
- Removed `ApproachScoring` and `ApproachRedundancy` from the `valid` map
- Updated doc comment on `Parse` to reflect two approaches

### `services/api-dashboard/finetune/confidence_benchmark_test.go`
- `allApproaches` slice now contains only `constraint` and `self_check`
- `writeResultsMD` table iteration updated to the same two approaches
- Updated `TestBenchmark` doc comment from "all 4" to "all"

## Validation

```
go build ./services/api-dashboard/...  → Success
go vet   ./services/api-dashboard/...  → No issues found
```
