# Fix: FileURL Validation in Creative Service

## Summary

Added `file_url` validation to `CreativeService.Create` and `CreativeService.Update`. Without this fix, creatives could be persisted with an empty or non-HTTP(S) `file_url`, breaking SDK delivery.

## Root Cause

`CreateCreativeRequest.FileURL` and `UpdateCreativeRequest.FileURL` had no validation guards. The field was accepted as-is and forwarded to the repository.

## Changes

### `services/api-dashboard/internal/service/creative.go`

- Added `"net/url"` import.
- Added package-level helper `validateFileURL(rawURL string) error` that:
  - Returns `ErrInvalidInput` when `rawURL` is empty.
  - Uses `url.ParseRequestURI` to reject structurally invalid strings.
  - Rejects any scheme other than `http` / `https`.
- Called `validateFileURL` in both `Create` and `Update`, placed after the `Type` check and before the `FileSizeBytes` check.

### Test fixture updates

Existing tests used `s3://bucket/…` URLs, which are now correctly rejected. All fixtures were updated to `https://cdn.example.com/…` URLs:

- `services/api-dashboard/internal/service/creative_test.go` — replaced all `s3://` fixtures.
- `services/api-dashboard/internal/handler/creative_test.go` — replaced all `s3://` fixtures.

### New test cases added (`creative_test.go`)

`TestCreativeService_Create`:
- `empty file_url` — expects `ErrInvalidInput`
- `invalid file_url scheme s3` — expects `ErrInvalidInput`
- `invalid file_url not a url` — expects `ErrInvalidInput`

`TestCreativeService_Update`:
- `empty file_url` — expects `ErrInvalidInput`
- `invalid file_url scheme s3` — expects `ErrInvalidInput`

## Verification

```
go build ./services/api-dashboard/...  → success
go test ./services/api-dashboard/...   → 344 passed, 0 failed
```
