# Go Handler Tests — Phase 2 Publisher Domain

Agent: go-test-writer
Stage: Test (handler layer)
Status: DONE — go test ./services/api-dashboard/... passed (223 tests, 0 failures)

## Files Created

| File | Tests |
|------|-------|
| `services/api-dashboard/internal/handler/publisher_app_test.go` | TestPublisherAppHandler_Create (6 cases), _GetByID (4), _List (3), _Update (6) |
| `services/api-dashboard/internal/handler/api_key_test.go` | TestAPIKeyHandler_Create (5 cases + plaintext/hash assertions), _ListByApp (4 + key_hash absence assertion), _Revoke (5) |
| `services/api-dashboard/internal/handler/publisher_rule_test.go` | TestPublisherRuleHandler_Create (13 cases), _GetByID (4), _ListByApp (4), _Update (6), _Delete (5) |

## Total New Handler Tests

42 table-driven test cases added across 3 files.

## Test Strategy

### Mock Pattern

Each test file declares a private mock struct implementing the corresponding repository interface via func fields. This matches the existing pattern from `organization_test.go` and `org_invite_test.go`:

```go
type mockPublisherAppRepoForHandler struct {
    insertFn      func(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error)
    getByIDFn     func(ctx context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error)
    getByBundleFn func(ctx context.Context, orgID uuid.UUID, bundleID string) (*model.PublisherApp, error)
    listByOrgFn   func(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]model.PublisherApp, int64, error)
    updateFn      func(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error)
}
```

Compile-time interface satisfaction verified via `var _ repository.XxxRepository = (*mockXxxRepoForHandler)(nil)`.

### Shared Helpers (in publisher_app_test.go)

The following helpers are shared across all three test files within the `handler` package:

- `marshalBody(t, body)` — marshals body (string passthrough for invalid JSON tests)
- `assertStatus(t, w, want)` — checks HTTP status code with body in error message
- `assertErrorCode(t, w, want)` — decodes response and checks `error.code`
- `withChiAppAndRuleID(r, appID, ruleID)` — injects `{id}` + `{ruleId}` into chi route context
- `withChiAppAndKeyID(r, appID, keyID)` — injects `{id}` + `{keyId}` into chi route context
- `injectMiddlewareContext(r, orgID, role)` — thin wrapper around `middleware.InjectTestContext`

The existing `injectAuthContext`, `withChiID`, `decodeRespBody` helpers from `organization_test.go` are reused (same package).

### Security Assertions (api_key_test.go)

The following invariants are explicitly asserted in handler tests:

1. **Create response MUST include `key` field** (plaintext returned exactly once)
2. **Create response MUST include `key_prefix` field**
3. **Create response MUST NOT include `key_hash` field** (tagged `json:"-"` on model.APIKey)
4. **List response items MUST NOT include `key` field** (model.APIKey has no plaintext field)
5. **List response items MUST NOT include `key_hash` field**
6. **List response items MUST include `key_prefix` field**
7. **Revoke response MUST NOT include `key_hash` field**

The `key` field is delivered via `apiKeyCreateResponse` struct (separate from `model.APIKey`) — the handler test confirms this separation holds end-to-end.

### Coverage per Handler

**PublisherAppHandler:**
- Create: valid, invalid JSON, empty name, invalid platform, duplicate bundle_id, repo error
- GetByID: found, invalid UUID, not found, cross-org (returns 404 due to org_id filter)
- List: with items, empty list, repo error
- Update: name update, is_active update, invalid UUID, invalid JSON, not found, empty name

**APIKeyHandler:**
- Create: valid (with key assertions), invalid app UUID, invalid JSON, empty name, repo error
- ListByApp: active only (default), include_revoked param, invalid UUID, repo error
- Revoke: valid (with response field assertions), invalid app UUID, invalid key UUID, not found, already revoked, repo error

**PublisherRuleHandler:**
- Create: blocklist, frequency_cap, geo_filter valid cases; invalid app UUID, invalid JSON, unknown type, freq_cap missing max_impressions, geo_filter invalid mode, geo_filter empty country_codes, repo error
- GetByID: found, invalid app UUID, invalid rule UUID, not found
- ListByApp: with items, empty, invalid app UUID, repo error
- Update: config update, is_active toggle, invalid app UUID, invalid rule UUID, invalid JSON, not found
- Delete: valid (with id assertion), invalid app UUID, invalid rule UUID, not found, repo error

## Verification

```
go test ./services/api-dashboard/internal/handler/... — 114 tests passed (72 existing + 42 new)
go test ./services/api-dashboard/...                  — 223 tests passed (all packages)
```
