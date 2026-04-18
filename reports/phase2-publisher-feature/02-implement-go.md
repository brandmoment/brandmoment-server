# Go Implementation — Phase 2 Publisher Domain

Agent: go-builder
Stage: Implement (Go layer)
Status: DONE — go build, go vet, go test all passed (145 tests)

## Files Created

### Models

| File | Description |
|------|-------------|
| `services/api-dashboard/internal/model/publisher_app.go` | PublisherApp struct |
| `services/api-dashboard/internal/model/api_key.go` | APIKey struct; KeyHash tagged with json:"-" so it never appears in API responses |
| `services/api-dashboard/internal/model/publisher_rule.go` | PublisherRule struct, 5 rule type constants, per-type config structs for validation |

### Repositories

| File | Description |
|------|-------------|
| `services/api-dashboard/internal/repository/publisher_app.go` | Interface + impl; Insert, GetByID, GetByBundleID, ListByOrg (calls Count query), Update |
| `services/api-dashboard/internal/repository/api_key.go` | Interface + impl; Insert, GetByID, ListByApp (activeOnly flag), Revoke |
| `services/api-dashboard/internal/repository/publisher_rule.go` | Interface + impl; Insert, GetByID, ListByApp (calls Count query), Update, Delete |

### Services

| File | Description |
|------|-------------|
| `services/api-dashboard/internal/service/publisher_app.go` | Create (bundle_id uniqueness check), GetByID, List (limit/offset clamping), Update (partial) |
| `services/api-dashboard/internal/service/api_key.go` | Provision (crypto/rand + sha256 + prefix), ListByApp, Revoke (already-revoked guard) |
| `services/api-dashboard/internal/service/publisher_rule.go` | Create, GetByID, List, Update, Delete; validateRuleType + validateRuleConfig per type |

### Handlers

| File | Description |
|------|-------------|
| `services/api-dashboard/internal/handler/publisher_app.go` | Create, List, GetByID, Update; parsePagination helper |
| `services/api-dashboard/internal/handler/api_key.go` | Create (returns plaintext once via apiKeyCreateResponse), List, Revoke |
| `services/api-dashboard/internal/handler/publisher_rule.go` | Create, List, GetByID, Update, Delete; parseAppIDParam helper |

### Tests

| File | Tests |
|------|-------|
| `services/api-dashboard/internal/service/publisher_app_test.go` | TestPublisherAppService_Create (6 cases), _GetByID (2), _List (3), _Update (4) |
| `services/api-dashboard/internal/service/api_key_test.go` | TestAPIKeyService_Provision (2 cases + hash/prefix assertions), _ListByApp (2), _Revoke (3) |
| `services/api-dashboard/internal/service/publisher_rule_test.go` | TestPublisherRuleService_Create (12 cases covering all 5 types + invalid configs), _List (3), _Update (3), _Delete (2) |

### Modified Files

| File | Change |
|------|--------|
| `services/api-dashboard/internal/router/router.go` | Added PublisherApp, APIKey, PublisherRule to Handlers struct; registered all routes with per-route RBAC via .With() |
| `services/api-dashboard/cmd/server/main.go` | Wired 3 new repo/service/handler chains into DI |

## Design Decisions

### Security
- `APIKey.KeyHash` is tagged `json:"-"` — never serialized in any response.
- `apiKeyCreateResponse` is a separate struct used only in the Create handler, explicitly carrying the `Key` (plaintext) field. All other responses use `model.APIKey` which hides the hash and has no plaintext field.
- `slog.InfoContext` never logs plaintext key or key_hash — only org_id, app_id, name, key_prefix.
- OTel spans do not record key plaintext in attributes.

### Multi-tenancy
- All repository methods take `orgID uuid.UUID` as an explicit parameter. Impossible to query without it (compile-time enforcement).
- `GetByBundleID` includes `org_id` filter — bundle_id uniqueness is per-org, not global.
- API keys and rules pass both `orgID` and `appID` to every query, enforcing double scoping.

### Pagination
- `parsePagination` in handler extracts limit/offset from query params with defaults (20/0).
- Service layer clamps limit to [1, 100] — handler can pass any value, service corrects it.

### RBAC in Router
- Used `.With(auth.RequireRole(...))` per-method rather than group-level middleware, because different HTTP methods on the same path need different roles (viewer for GET, editor for POST, admin for DELETE).

### Rule Validation
- `validateRuleConfig` deserializes config into the appropriate struct for structural validation.
- `frequency_cap` explicitly checks `max_impressions > 0` and `window_seconds > 0` (zero values are valid JSON but semantically invalid).
- `geo_filter` and `platform_filter` validate mode as "include|exclude" and require non-empty arrays.

## Verification

```
go build ./services/api-dashboard/...  — SUCCESS
go vet ./services/api-dashboard/...    — no issues
go test ./services/api-dashboard/...   — 145 tests passed (92 existing + 53 new)
```
