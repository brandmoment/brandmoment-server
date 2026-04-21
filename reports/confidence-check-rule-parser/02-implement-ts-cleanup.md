# TS Cleanup: Remove scoring + redundancy approaches

## Summary

Removed `scoring` and `redundancy` confidence approaches from the dashboard UI.
Only `constraint` and `self_check` are now exposed.

## Files Changed

### `apps/dashboard/types/rule-parser.ts`
- `ApproachName` union narrowed from 4 values to `"constraint" | "self_check"`
- Removed `ScoringApproachResult` interface (had `confidence`, `reasoning` fields)
- Removed `RedundancyApproachResult` interface (had `agreement_count`, `total_runs` fields)
- `ApproachResult` union now covers only `ConstraintApproachResult | SelfCheckApproachResult`
- `ApproachResults` object type now has only `constraint?` and `self_check?` fields

### `apps/dashboard/components/publisher/RuleParserPage.tsx`
- `ALL_APPROACHES` array trimmed to 2 entries: Constraint + Self-Check
- `ApproachRow` details logic: removed scoring branch (`confidence * 100`, `reasoning`) and redundancy branch (`agreement_count/total_runs`); self_check branch retained
- `labelMap` inside `ApproachRow` reduced to 2 entries
- `useState` default set for `selectedApproaches` changed from all 4 to `["constraint", "self_check"]`

### `apps/dashboard/hooks/useParseRule.ts`
- No changes required — hook passes through whatever approaches array the caller provides, no hardcoded defaults

## Typecheck Result

`pnpm typecheck` produced 0 errors in modified files.

Pre-existing auth-related errors remain unchanged (signIn/signUp/signOut resolution, OrgSwitcher organization property) — not introduced by this change and not in scope.

## Dead Code Removed

No dead code left. All branches referencing scoring or redundancy have been deleted rather than commented out.
