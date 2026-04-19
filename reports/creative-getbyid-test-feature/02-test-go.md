# Test Report: CreativeHandler.GetByID

## Summary

Added `GetByID` handler method and full test coverage for `TestCreativeHandler_GetByID`.

## Source Changes

### `services/api-dashboard/internal/handler/creative.go`
Added `GetByID` method. Takes `{id}` (campaignID) and `{creativeId}` from chi URL params, delegates to `service.CreativeService.GetByID`, responds 200 with creative or propagates service errors via `handleServiceError`.

### `services/api-dashboard/internal/router/router.go`
Wired `GET /v1/campaigns/{id}/creatives/{creativeId}` to `h.Creative.GetByID` with `viewer+` role requirement.

### `services/api-dashboard/internal/handler/creative_test.go`
- Added `chi` import
- Added `withChiCampaignAndCreativeID` helper (injects both `{id}` and `{creativeId}` into chi route context)
- Added `TestCreativeHandler_GetByID` with 6 table-driven cases

## Tests Written

| File | Function | Cases |
|------|----------|-------|
| `services/api-dashboard/internal/handler/creative_test.go` | `TestCreativeHandler_GetByID` | 6 |

### Test Cases

| Case | Scenario | Expected |
|------|----------|----------|
| found creative returns 200 | Valid campaign + creative IDs, repo returns creative | 200 OK |
| creative not found returns 404 | Repo returns `ErrNotFound` | 404 NOT_FOUND |
| creative belongs to different org returns 404 | Cross-org lookup; repo enforces org_id filter, returns `ErrNotFound` | 404 NOT_FOUND |
| invalid campaign UUID in URL returns 400 | `{id}` = "not-a-uuid" | 400 INVALID_ID |
| invalid creative UUID in URL returns 400 | `{creativeId}` = "not-a-uuid" | 400 INVALID_ID |
| repo error returns 500 | Repo returns `context.DeadlineExceeded` | 500 INTERNAL_ERROR |

## Test Results

```
Go test: 7 passed in 1 packages
Go build: Success
Go vet: No issues found
```

(7 = 6 new GetByID cases + 1 existing suite pass; full handler package ran clean)

## Bugs Discovered

None.
