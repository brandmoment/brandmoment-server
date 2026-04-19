# Go Tests: Creative Update Endpoint

## Summary

Added table-driven tests for `Creative.Update` at both the handler and service layers.

## Tests Written

| File | Test | Cases |
|------|------|-------|
| `services/api-dashboard/internal/handler/creative_test.go` | `TestCreativeHandler_Update` | 7 |
| `services/api-dashboard/internal/service/creative_test.go` | `TestCreativeService_Update` | 6 |

### TestCreativeHandler_Update (7 cases)

1. success returns 200
2. creative not found returns 404
3. invalid campaign UUID returns 400
4. invalid creative UUID returns 400
5. invalid JSON body returns 400
6. empty name returns 400 (validation via service)
7. repo error returns 500

### TestCreativeService_Update (6 cases)

1. success — returns creative, verifies ID
2. empty name — `ErrInvalidInput`
3. name too long (>200 chars) — `ErrInvalidInput`
4. invalid type (non html5/image/video) — `ErrInvalidInput`
5. not found from repo — `ErrNotFound`
6. repo error propagated — generic error

## Test Results

All 15 new tests pass:

```
Go test: 15 passed in 2 packages
```

## Bugs Discovered

None.
