# Implement: Extract parsePagination and Move handleServiceError

## Summary

Both issues #8 and #9 resolved in a single pass. Created
`services/api-dashboard/internal/handler/helpers.go` as the canonical home for
package-level handler utilities.

## Changes

### Created
- `services/api-dashboard/internal/handler/helpers.go`
  - `parsePagination(r *http.Request) (limit, offset int32)` — moved from `publisher_app.go`
  - `handleServiceError(w http.ResponseWriter, err error)` — moved from `organization.go`
  - Imports: `errors`, `net/http`, `strconv`, `httputil`, `model`

### Modified
- `services/api-dashboard/internal/handler/publisher_app.go`
  - Removed `parsePagination` definition and its `strconv` import (no longer needed in this file)
- `services/api-dashboard/internal/handler/organization.go`
  - Removed `handleServiceError` definition and its `errors` + `model` imports (no longer needed in this file)

## No-change guarantee

Function signatures and behavior are identical. All callers
(`campaign.go`, `publisher_app.go`, `publisher_rule.go`, and all organization/other
handlers) call the same package-level names — no call sites required updating.

## Verification

```
go build ./services/api-dashboard/...  — SUCCESS
go test ./services/api-dashboard/...   — 324 passed, 0 failed (9 packages)
```
