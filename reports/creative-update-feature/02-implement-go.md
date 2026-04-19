# Implementation: Creative Update Endpoint

## Status: Done

## Changes

### Repository (`services/api-dashboard/internal/repository/creative.go`)
- Added `UpdateCreativeParams` struct (Name, Type, FileURL, FileSizeBytes, PreviewURL, IsActive) to keep domain types separate from sqlc types.
- Extended `CreativeRepository` interface with `Update(ctx, orgID, campaignID, id, params) (*model.Creative, error)`.
- Implemented `(creativeRepo).Update` — calls `q.UpdateCreative`, maps `pgx.ErrNoRows` → `model.ErrNotFound`, wraps other errors with `fmt.Errorf("update creative: %w", err)`.

### Service (`services/api-dashboard/internal/service/creative.go`)
- Added `UpdateCreativeRequest` struct (Name, Type, FileURL, FileSizeBytes, PreviewURL, IsActive).
- Added `Update(ctx, orgID, campaignID, id, req)` method with OTel span (`CreativeService.Update`) and `defer span.End()`.
- Same validation as `Create`: name 1–200 chars, type in {html5, image, video}, FileSizeBytes > 0 if set.
- Calls `creativeRepo.Update` with `repository.UpdateCreativeParams`.

### Handler (`services/api-dashboard/internal/handler/creative.go`)
- Added `Update(w, r)` method.
- Parses `campaignID` from chi URL param `{id}`, `creativeID` from `{creativeId}`.
- Decodes JSON body into `service.UpdateCreativeRequest` via `json.NewDecoder`.
- Calls `service.Update` with `orgID` from `middleware.OrgIDFromContext` (never from body).
- Responds 200 with updated creative or delegates to `handleServiceError`.

### Router (`services/api-dashboard/internal/router/router.go`)
- Added `r.With(auth.RequireRole("editor", "admin", "owner")).Put("/", h.Creative.Update)` inside the `/{creativeId}` route group.

### Test mocks (existing test files updated to satisfy new interface)
- `services/api-dashboard/internal/service/creative_test.go` — added `updateFn` field and `Update` method to `mockCreativeRepo`.
- `services/api-dashboard/internal/handler/creative_test.go` — added `updateFn` field and `Update` method to `mockCreativeRepoForHandler`.

## Verification

```
go build ./services/api-dashboard/...  ✓
go vet ./services/api-dashboard/...    ✓
```
