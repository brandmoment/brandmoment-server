# Implement: Consolidate UUID Conversion Helpers

## Summary

Extracted `uuidToPgtype` and `pgtypeToUUID` from `organization.go` into a dedicated `converters.go` file in the same package. No behavior changes — same package means all callers resolve the functions without any import updates.

## Files Changed

### Created
- `services/api-dashboard/internal/repository/converters.go`
  - Contains `uuidToPgtype(uuid.UUID) pgtype.UUID` and `pgtypeToUUID(pgtype.UUID) uuid.UUID`
  - Imports: `github.com/google/uuid`, `github.com/jackc/pgx/v5/pgtype`

### Modified
- `services/api-dashboard/internal/repository/organization.go`
  - Removed the two helper function definitions (lines 73-79 in original)
  - All imports retained: `pgtype` still used for `pgtype.Timestamptz` and `pgtype.UUID`; `uuid` used in interface/method signatures

## Callers (unchanged)

The helpers are used across all 8 repository files in the package:

| File | `uuidToPgtype` calls | `pgtypeToUUID` calls |
|---|---|---|
| `organization.go` | 4 | 1 |
| `api_key.go` | 7 | 3 |
| `campaign.go` | 8 | 2 |
| `creative.go` | 7 | 3 |
| `org_invite.go` | 2 | 2 |
| `publisher_app.go` | 8 | 2 |
| `publisher_rule.go` | 11 | 3 |
| `user.go` | 2 | 1 |

No call sites needed updating — same package.

## Verification

```
go build ./services/api-dashboard/...  → Success
go test ./services/api-dashboard/...   → 324 passed in 9 packages
```
