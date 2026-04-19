# Fix: Unchecked json.Encode Error in httputil Response Helpers

## File Modified

`services/api-dashboard/internal/httputil/response.go`

## Root Cause

`json.NewEncoder(w).Encode(...)` return values were silently discarded on lines 24 and 31. If encoding fails (e.g., an unencodable type passed as `data`), the error went unnoticed and no diagnostic was emitted, making failures invisible in logs.

## Changes

1. Added `"log/slog"` to imports.
2. `RespondJSON`: wrapped `Encode` call in error check; logs `slog.Error("failed to encode response", ...)` on failure.
3. `RespondError`: wrapped `Encode` call in error check; logs `slog.Error("failed to encode error response", ...)` on failure.

`slog.Error` (without context) is used because neither helper accepts a `context.Context` parameter and changing the signature would break all callers. Since `WriteHeader` is called before `Encode`, the HTTP status is already sent — logging is the only viable recovery action.

## Verification

```
go build ./services/api-dashboard/...  → OK
go vet ./services/api-dashboard/...    → OK
```
