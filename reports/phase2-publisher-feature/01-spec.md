# Phase 2: Publisher Domain + Dashboard Publisher Pages

## 1. Context & Problem

BrandMoment is a multi-tenant ad network platform serving three org types: admin, publisher, brand. Phase 1 delivered identity (users, org_memberships, org_invites) and OpenAPI scaffolding. Phase 2 builds the publisher domain — the first revenue-generating domain that lets publisher orgs register mobile/web apps, provision SDK API keys, and configure ad filtering rules. Without this domain, publishers cannot integrate the SDK or receive ad revenue.

Current system state:
- 4 migrations in place: organizations, users, org_memberships, org_invites
- JWT auth middleware working (HMAC, migration to JWKS planned)
- Router at `/v1` with organizations CRUD, `/me`, org invites
- Handler/service/repository layering established and tested (92 unit tests passing)
- OpenAPI spec at `packages/proto/dashboard.yaml` (Phase 1 additions pending)
- Dashboard: NOT STARTED — scaffold with auth pages and layout shell exists per docs

## 2. Goals & Non-Goals

### Goals
- Create publisher_apps, api_keys, publisher_rules tables with migrations
- Implement full CRUD for publisher apps under org_id isolation
- Implement API key provisioning (shown once in plaintext) and revocation
- Implement publisher rules CRUD with JSONB config per rule type
- Add pagination (limit/offset) to all list endpoints
- Add publisher pages to the dashboard: /apps list, /apps/:id detail with tabs
- Maintain existing auth/multi-tenancy invariants on all new endpoints

### Non-Goals
- SDK hot-path consumption of API keys (belongs to api-sdk service)
- Argon2 hashing of API keys (SHA-256 used per decision; argon2 is a future upgrade)
- Analytics/metrics for publisher apps (Phase 4)
- Brand/campaign domain (Phase 3)
- Real-time rule evaluation engine (future v2)

## 3. User Stories

- As a publisher I want to register my mobile/web app so that I can integrate the BrandMoment SDK.
- As a publisher I want to provision an API key for my app so that my SDK can authenticate with the platform.
- As a publisher I want to revoke an API key so that I can rotate credentials without deleting the app.
- As a publisher I want to configure blocklist/allowlist rules for my app so that I can control which ads are shown.
- As a publisher I want to set frequency caps and geo/platform filters so that I can protect user experience.
- As a viewer I want to see my org's apps and their rules without being able to modify them.

## 4. Scope

### In Scope
- 3 SQL migrations (publisher_apps, api_keys, publisher_rules)
- sqlc queries for all 3 entities
- Go model, repository, service, handler, tests for publisher_apps, api_keys, publisher_rules
- Router registration for all new endpoints
- OpenAPI spec additions in `packages/proto/dashboard.yaml`
- TypeScript dashboard pages: /apps, /apps/:id (Overview / API Keys / Rules tabs)
- API key create/revoke UI with one-time plaintext display
- Rules editor UI for all 5 rule types

### Out of Scope
- SDK-side key validation (api-sdk service)
- Publisher analytics pages (Phase 4)
- App deletion (soft-delete via is_active=false only — no hard delete endpoint)

## 5. Functional Requirements

- FR-1: publisher_apps are scoped to an org. Every query filters by org_id from JWT context.
- FR-2: api_keys store key_hash (SHA-256) and key_prefix (first 8 chars). The plaintext key is returned exactly once on creation and never stored.
- FR-3: API key revocation sets is_revoked=true and revoked_at=now(). Revoked keys are excluded from active key queries.
- FR-4: publisher_rules support 5 types: blocklist, allowlist, frequency_cap, geo_filter, platform_filter. Config schema is JSONB; validation of config shape is done in the service layer.
- FR-5: List endpoints for publisher_apps and publisher_rules support limit (default 20, max 100) and offset query params.
- FR-6: api_keys list for an app returns only non-revoked keys by default; a `?include_revoked=true` param returns all.
- FR-7: RBAC: viewer role can call GET endpoints. editor, admin, owner can call POST/PUT. Only admin and owner can call DELETE (revoke key) and deactivate app.
- FR-8: All mutation endpoints validate org_id from JWT context — never trust org_id in request body.
- FR-9: OTel span created per service method; errors recorded with span.RecordError.
- FR-10: Dashboard shows a one-time modal with full API key value after creation; subsequent views show key_prefix + "..." only.

## 6. API Changes

### Existing Endpoints Affected
- `internal/router/router.go` — add PublisherApp, APIKey, PublisherRule handlers to Handlers struct and register routes
- `packages/proto/dashboard.yaml` — add all new endpoint definitions

### New Endpoints

#### Publisher Apps

**POST /v1/publisher-apps**
- Auth: JWT + editor/admin/owner
- Request:
  ```json
  {
    "name": "string (required, 1-100 chars)",
    "platform": "ios | android | web",
    "bundle_id": "string (required, e.g. com.example.app)"
  }
  ```
- Response 201:
  ```json
  {
    "data": {
      "id": "uuid",
      "org_id": "uuid",
      "name": "string",
      "platform": "ios | android | web",
      "bundle_id": "string",
      "is_active": true,
      "created_at": "RFC3339",
      "updated_at": "RFC3339"
    }
  }
  ```
- Errors: 400 INVALID_INPUT, 401 UNAUTHORIZED, 403 FORBIDDEN

**GET /v1/publisher-apps**
- Auth: JWT + viewer/editor/admin/owner
- Query params: `limit` (int, default 20, max 100), `offset` (int, default 0)
- Response 200:
  ```json
  {
    "data": {
      "items": [ { ...PublisherApp } ],
      "total": 42,
      "limit": 20,
      "offset": 0
    }
  }
  ```

**GET /v1/publisher-apps/{id}**
- Auth: JWT + viewer/editor/admin/owner
- Path param: `id` (UUID)
- Response 200: `{"data": { ...PublisherApp }}`
- Errors: 400 INVALID_ID, 404 NOT_FOUND

**PUT /v1/publisher-apps/{id}**
- Auth: JWT + editor/admin/owner
- Request: same shape as POST (all fields optional; omitted fields unchanged)
  ```json
  {
    "name": "string (optional)",
    "is_active": "bool (optional)"
  }
  ```
- Response 200: `{"data": { ...PublisherApp }}`
- Errors: 400 INVALID_INPUT, 400 INVALID_ID, 404 NOT_FOUND, 403 FORBIDDEN

#### API Keys

**POST /v1/publisher-apps/{id}/api-keys**
- Auth: JWT + editor/admin/owner
- Request:
  ```json
  { "name": "string (required, label for the key)" }
  ```
- Response 201:
  ```json
  {
    "data": {
      "id": "uuid",
      "org_id": "uuid",
      "app_id": "uuid",
      "name": "string",
      "key": "bm_<random_32_bytes_hex>",
      "key_prefix": "bm_a1b2c3",
      "is_revoked": false,
      "created_at": "RFC3339"
    }
  }
  ```
  Note: `key` field is present ONLY in this creation response. All subsequent reads return key_prefix only.
- Errors: 400 INVALID_INPUT, 404 NOT_FOUND (app not found in org)

**GET /v1/publisher-apps/{id}/api-keys**
- Auth: JWT + viewer/editor/admin/owner
- Query params: `include_revoked` (bool, default false)
- Response 200:
  ```json
  {
    "data": {
      "items": [
        {
          "id": "uuid",
          "org_id": "uuid",
          "app_id": "uuid",
          "name": "string",
          "key_prefix": "bm_a1b2c3",
          "is_revoked": false,
          "created_at": "RFC3339",
          "revoked_at": null
        }
      ]
    }
  }
  ```

**DELETE /v1/publisher-apps/{id}/api-keys/{keyId}**
- Auth: JWT + admin/owner
- Sets is_revoked=true, revoked_at=now()
- Response 200:
  ```json
  { "data": { "id": "uuid", "revoked_at": "RFC3339" } }
  ```
- Errors: 400 INVALID_ID, 404 NOT_FOUND, 403 FORBIDDEN

#### Publisher Rules

**GET /v1/publisher-apps/{id}/rules**
- Auth: JWT + viewer/editor/admin/owner
- Query params: `limit` (default 20, max 100), `offset` (default 0)
- Response 200:
  ```json
  {
    "data": {
      "items": [
        {
          "id": "uuid",
          "org_id": "uuid",
          "app_id": "uuid",
          "type": "blocklist | allowlist | frequency_cap | geo_filter | platform_filter",
          "config": { ... },
          "is_active": true,
          "created_at": "RFC3339",
          "updated_at": "RFC3339"
        }
      ],
      "total": 5,
      "limit": 20,
      "offset": 0
    }
  }
  ```

**POST /v1/publisher-apps/{id}/rules**
- Auth: JWT + editor/admin/owner
- Request:
  ```json
  {
    "type": "blocklist | allowlist | frequency_cap | geo_filter | platform_filter",
    "config": { ... }
  }
  ```
- Config schemas by type:
  - blocklist/allowlist: `{ "domains": ["string"], "bundle_ids": ["string"] }`
  - frequency_cap: `{ "max_impressions": int, "window_seconds": int }`
  - geo_filter: `{ "mode": "include | exclude", "country_codes": ["US", "DE"] }`
  - platform_filter: `{ "mode": "include | exclude", "platforms": ["ios", "android", "web"] }`
- Response 201: `{"data": { ...PublisherRule }}`
- Errors: 400 INVALID_INPUT (unknown type, invalid config shape)

**GET /v1/publisher-apps/{id}/rules/{ruleId}**
- Auth: JWT + viewer/editor/admin/owner
- Response 200: `{"data": { ...PublisherRule }}`
- Errors: 400 INVALID_ID, 404 NOT_FOUND

**PUT /v1/publisher-apps/{id}/rules/{ruleId}**
- Auth: JWT + editor/admin/owner
- Request: `{ "config": { ... }, "is_active": bool }` (partial update)
- Response 200: `{"data": { ...PublisherRule }}`
- Errors: 400 INVALID_INPUT, 404 NOT_FOUND

**DELETE /v1/publisher-apps/{id}/rules/{ruleId}**
- Auth: JWT + admin/owner
- Hard delete (rules are config, not business records)
- Response 200: `{"data": {"id": "uuid"}}`
- Errors: 400 INVALID_ID, 404 NOT_FOUND, 403 FORBIDDEN

## 7. Data Model

### Migration 000005: publisher_apps

```sql
-- 000005_create_publisher_apps.up.sql
CREATE TABLE IF NOT EXISTS publisher_apps (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    platform   TEXT NOT NULL CHECK (platform IN ('ios', 'android', 'web')),
    bundle_id  TEXT NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_publisher_apps_org_id ON publisher_apps (org_id);
CREATE INDEX idx_publisher_apps_org_id_is_active ON publisher_apps (org_id, is_active);

-- 000005_create_publisher_apps.down.sql
DROP TABLE IF EXISTS publisher_apps;
```

### Migration 000006: api_keys

```sql
-- 000006_create_api_keys.up.sql
CREATE TABLE IF NOT EXISTS api_keys (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    app_id     UUID NOT NULL REFERENCES publisher_apps(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    key_hash   TEXT NOT NULL UNIQUE,
    key_prefix TEXT NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_app_id ON api_keys (app_id);
CREATE INDEX idx_api_keys_org_id ON api_keys (org_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys (key_hash);

-- 000006_create_api_keys.down.sql
DROP TABLE IF EXISTS api_keys;
```

### Migration 000007: publisher_rules

```sql
-- 000007_create_publisher_rules.up.sql
CREATE TABLE IF NOT EXISTS publisher_rules (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    app_id     UUID NOT NULL REFERENCES publisher_apps(id) ON DELETE CASCADE,
    type       TEXT NOT NULL CHECK (type IN ('blocklist', 'allowlist', 'frequency_cap', 'geo_filter', 'platform_filter')),
    config     JSONB NOT NULL DEFAULT '{}',
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_publisher_rules_app_id ON publisher_rules (app_id);
CREATE INDEX idx_publisher_rules_org_id ON publisher_rules (org_id);
CREATE INDEX idx_publisher_rules_app_id_is_active ON publisher_rules (app_id, is_active);

-- 000007_create_publisher_rules.down.sql
DROP TABLE IF EXISTS publisher_rules;
```

### Multi-tenancy

All three tables are sub-resources of organizations. Every SQL query in sqlc files MUST include `WHERE org_id = @org_id` (and additionally `AND app_id = @app_id` for api_keys and publisher_rules). Org_id is never taken from request body — only from JWT context via `middleware.OrgIDFromContext(ctx)`.

## 8. UI Changes

### Affected Pages / Navigation

- Sidebar (publisher org): add "Apps" nav item linking to `/apps`
- Sidebar currently: per docs, publisher sidebar shows Apps, Rules, Analytics

### New Pages and Components

**`/apps` — Publisher App List**
- File: `apps/dashboard/app/(dashboard)/apps/page.tsx`
- Server Component: fetch session, pass to client
- Client Component: `PublisherAppsList.tsx`
  - TanStack Table with columns: Name, Platform, Bundle ID, Status, Created At, Actions
  - Sorting: name, created_at
  - Filtering: platform (select), is_active (toggle)
  - Pagination: limit/offset, page size selector (20/50/100)
  - "New App" button → opens `CreateAppDialog`
- `CreateAppDialog.tsx`: React Hook Form + zod, fields: name, platform (select), bundle_id
- Loading state: shadcn Skeleton rows

**`/apps/:id` — Publisher App Detail**
- File: `apps/dashboard/app/(dashboard)/apps/[id]/page.tsx`
- Tabs (shadcn Tabs): Overview, API Keys, Rules

- **Overview tab** (`AppOverviewTab.tsx`):
  - App metadata: name, platform, bundle_id, is_active badge, created_at
  - Edit form inline (editor+ only): name, is_active toggle
  - "Save" button, optimistic update via TanStack Query mutation

- **API Keys tab** (`AppApiKeysTab.tsx`):
  - List of keys: name, prefix, created_at, status badge (active/revoked)
  - "Create Key" button → `CreateApiKeyDialog`
  - After creation: `ApiKeyRevealModal` shows full key in monospace with copy button and explicit "I have copied this key" confirmation; key is never retrievable again
  - Revoke button (admin/owner only) per key row → confirmation dialog → DELETE call

- **Rules tab** (`AppRulesTab.tsx`):
  - List of rules: type, is_active toggle, config summary, actions
  - "Add Rule" button → `RuleEditorDialog`
  - `RuleEditorDialog.tsx`: type selector changes form fields:
    - blocklist/allowlist: textarea for domains, textarea for bundle_ids (one per line)
    - frequency_cap: number inputs for max_impressions and window_seconds
    - geo_filter: mode radio (include/exclude), multi-select country codes
    - platform_filter: mode radio, checkbox group (ios, android, web)
  - Inline is_active toggle per rule row (editor+ only)
  - Delete rule button (admin/owner only)

### Hooks

- `usePublisherApps.ts` — `GET /v1/publisher-apps` with pagination params
- `usePublisherApp.ts` — `GET /v1/publisher-apps/:id`
- `useCreatePublisherApp.ts` — `POST /v1/publisher-apps`
- `useUpdatePublisherApp.ts` — `PUT /v1/publisher-apps/:id`
- `useApiKeys.ts` — `GET /v1/publisher-apps/:id/api-keys`
- `useCreateApiKey.ts` — `POST /v1/publisher-apps/:id/api-keys`
- `useRevokeApiKey.ts` — `DELETE /v1/publisher-apps/:id/api-keys/:keyId`
- `usePublisherRules.ts` — `GET /v1/publisher-apps/:id/rules`
- `useCreateRule.ts` — `POST /v1/publisher-apps/:id/rules`
- `useUpdateRule.ts` — `PUT /v1/publisher-apps/:id/rules/:ruleId`
- `useDeleteRule.ts` — `DELETE /v1/publisher-apps/:id/rules/:ruleId`

All hooks wrap `openapi-fetch` client with `Authorization: Bearer` + `X-Org-ID` headers from session.

## 9. State & Flows

### Happy Path: Create App + Provision Key

1. Publisher navigates to /apps → sees empty list
2. Clicks "New App" → fills name="MyApp", platform=ios, bundle_id=com.example.myapp → POST /v1/publisher-apps → 201
3. App appears in list. Publisher clicks into app detail.
4. Clicks "API Keys" tab → empty list
5. Clicks "Create Key" → fills name="Production" → POST /v1/publisher-apps/{id}/api-keys → 201
6. Modal shows full key (`bm_...`) with copy button. Publisher copies key. Clicks "I have copied this key".
7. Modal closes. Key list shows "Production" with prefix and Active badge.

### Happy Path: Configure Rule

1. Publisher navigates to /apps/:id → Rules tab
2. Clicks "Add Rule" → selects type=geo_filter → sets mode=exclude, country_codes=[CN, RU]
3. POST /v1/publisher-apps/{id}/rules → 201 → rule appears in list with config summary "Exclude: CN, RU"
4. Publisher toggles is_active=false → PUT /v1/publisher-apps/{id}/rules/{ruleId} → 200

### Edge Cases

- Creating app with duplicate bundle_id within same org: service layer returns ErrInvalidInput with message "bundle_id already exists for this org". (Note: no UNIQUE constraint on bundle_id+org_id at DB level; uniqueness enforced in service via GetByBundleID query.)
- Revoking already-revoked key: service returns ErrInvalidInput "key already revoked"
- Accessing /apps/:id for app that belongs to different org: repository returns ErrNotFound (org_id filter ensures cross-org data is invisible)
- Limit param > 100: service clamps to 100 and proceeds (no error)
- Creating rule with invalid config shape (e.g., frequency_cap missing max_impressions): service validates config JSON structure and returns ErrInvalidInput

### Error Handling

All errors flow through `handleServiceError` in handler layer (existing pattern):
- model.ErrNotFound → 404 NOT_FOUND
- model.ErrInvalidInput → 400 INVALID_INPUT
- model.ErrUnauthorized → 401 UNAUTHORIZED
- other → 500 INTERNAL_ERROR

Dashboard: all API errors surface as sonner toast notifications with the error.message from response body.

## 10. Non-Functional Requirements

### Performance
- List endpoints: response time < 50ms at p99 for up to 10k apps per org (index on org_id ensures this)
- API key provisioning: crypto/rand key generation + sha256 hash must complete < 5ms

### Security
- API key plaintext MUST NOT be logged (no slog.InfoContext with key value)
- API key MUST NOT appear in OTel span attributes
- key_hash stored as hex-encoded SHA-256: `fmt.Sprintf("%x", sha256.Sum256([]byte(plaintext)))`
- key_prefix is first 8 chars of plaintext key (including "bm_" prefix): safe to log and display
- Every sub-resource query MUST include org_id from context — repository interface enforces this at the function signature level (all methods take orgID uuid.UUID as first data param after ctx)
- RBAC per endpoint enforced via `auth.RequireRole` middleware grouping (see Section 6)

### Observability
- OTel spans: `PublisherAppService.Create`, `PublisherAppService.GetByID`, `PublisherAppService.List`, `PublisherAppService.Update`, `APIKeyService.Provision`, `APIKeyService.Revoke`, `APIKeyService.ListByApp`, `PublisherRuleService.Create`, `PublisherRuleService.GetByID`, `PublisherRuleService.List`, `PublisherRuleService.Update`, `PublisherRuleService.Delete`
- slog.InfoContext on all service method entry points with org_id, app_id (never key plaintext)

## 11. Dependencies

### Services Affected
- `services/api-dashboard` — new handlers, services, repositories, model files; router update
- `packages/shared-domain` — 3 new migration files, 3 new sqlc query files, sqlc regeneration required
- `packages/proto/dashboard.yaml` — new endpoint definitions
- `apps/dashboard` — new pages, components, hooks

### External APIs
- None. API key hashing uses stdlib `crypto/sha256`, key generation uses `crypto/rand`.

### Infrastructure
- No new infrastructure. Postgres 17 handles JSONB natively.
- No new docker-compose services required.

## 12. Testing Strategy

### Unit Tests (Go)

Table-driven tests for all service methods, following existing `TestOrganizationService_*` pattern:

- `TestPublisherAppService_Create` — valid input, empty name, invalid platform, duplicate bundle_id
- `TestPublisherAppService_GetByID` — found, not found, wrong org
- `TestPublisherAppService_List` — default pagination, custom limit, limit clamped to 100
- `TestPublisherAppService_Update` — valid partial update, not found
- `TestAPIKeyService_Provision` — valid, app not found in org
- `TestAPIKeyService_Revoke` — valid, already revoked, key not in org
- `TestAPIKeyService_ListByApp` — active only (default), include revoked
- `TestPublisherRuleService_Create` — each rule type with valid config, invalid config shape, unknown type
- `TestPublisherRuleService_List` — pagination
- `TestPublisherRuleService_Update` — valid, not found
- `TestPublisherRuleService_Delete` — valid, not found

Mock pattern: struct with func fields per existing convention (no testify/gomock):
```go
type mockPublisherAppRepo struct {
    insertFn    func(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error)
    getByIDFn   func(ctx context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error)
    listByOrgFn func(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]model.PublisherApp, int64, error)
    updateFn    func(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error)
}
```

### Integration Tests
- Not in scope for Phase 2 (no integration test infrastructure exists yet)

### Manual Verification Steps
1. Run `make infra-up`, apply migrations via `migrate -path infra/migrations -database $DATABASE_URL up`
2. Run `go build ./...` and `go vet ./...` in services/api-dashboard
3. Run `go test ./...` — all 92 existing + new tests pass
4. Run `sqlc generate` in packages/shared-domain — no errors
5. POST /v1/publisher-apps with valid JWT → 201, org_id in response matches X-Org-ID header
6. GET /v1/publisher-apps with different org's JWT → returns empty list (not 403, not cross-org data)
7. POST api-key → full key in response; GET api-keys → only prefix in response
8. DELETE api-key with viewer role JWT → 403 FORBIDDEN
9. PUT rule config → 200 with updated config
10. Dashboard: /apps loads list; create app dialog submits; app detail tabs render; key reveal modal shows once

## 13. Acceptance Criteria

- AC-1: `GET /v1/publisher-apps` returns only apps belonging to the org in the X-Org-ID header. A request with a different org's JWT returns 0 items, not the other org's apps.
- AC-2: `POST /v1/publisher-apps/{id}/api-keys` response contains a `key` field with the full plaintext. A subsequent `GET /v1/publisher-apps/{id}/api-keys` for the same key does NOT contain a `key` field — only `key_prefix`.
- AC-3: `DELETE /v1/publisher-apps/{id}/api-keys/{keyId}` with a viewer-role JWT returns 403. With admin/owner JWT returns 200 and subsequent GET shows is_revoked=true.
- AC-4: `POST /v1/publisher-apps/{id}/rules` with type=frequency_cap and missing `max_impressions` in config returns 400 INVALID_INPUT.
- AC-5: `GET /v1/publisher-apps?limit=200` returns at most 100 results (limit clamped).
- AC-6: `go test ./...` in services/api-dashboard passes with 0 failures, covering all new service methods.
- AC-7: `sqlc generate` in packages/shared-domain completes without errors after adding the 3 new query files.
- AC-8: Dashboard /apps page renders a TanStack Table with sorting by name and created_at, and platform filter.
- AC-9: Dashboard API key reveal modal displays the full key exactly once. After closing and reopening the API Keys tab, only key_prefix is visible.
- AC-10: Dashboard rules editor renders distinct form fields per rule type (geo_filter shows country code multi-select; frequency_cap shows numeric inputs).

## 14. Implementation Order

All implementation MUST follow this order to respect dependencies:

### Step 1: SQL (blocking for all Go work)

1. `infra/migrations/000005_create_publisher_apps.up.sql` + `.down.sql`
2. `infra/migrations/000006_create_api_keys.up.sql` + `.down.sql`
3. `infra/migrations/000007_create_publisher_rules.up.sql` + `.down.sql`
4. `packages/shared-domain/queries/publisher_apps.sql`
5. `packages/shared-domain/queries/api_keys.sql`
6. `packages/shared-domain/queries/publisher_rules.sql`
7. Run `sqlc generate` → generates `packages/shared-domain/db/publisher_apps.sql.go`, `api_keys.sql.go`, `publisher_rules.sql.go`

### Step 2: Go (parallel once SQL is done)

For each entity (publisher_apps, api_keys, publisher_rules) in this order per entity:

1. `services/api-dashboard/internal/model/publisher_app.go` — PublisherApp struct, sentinel errors
2. `services/api-dashboard/internal/model/api_key.go` — APIKey struct
3. `services/api-dashboard/internal/model/publisher_rule.go` — PublisherRule struct, RuleConfig interface, per-type config structs
4. `services/api-dashboard/internal/repository/publisher_app.go` — interface + impl
5. `services/api-dashboard/internal/repository/api_key.go` — interface + impl
6. `services/api-dashboard/internal/repository/publisher_rule.go` — interface + impl
7. `services/api-dashboard/internal/service/publisher_app.go` — business logic + OTel
8. `services/api-dashboard/internal/service/api_key.go` — key generation, hashing, revocation
9. `services/api-dashboard/internal/service/publisher_rule.go` — config validation per type
10. `services/api-dashboard/internal/service/publisher_app_test.go`
11. `services/api-dashboard/internal/service/api_key_test.go`
12. `services/api-dashboard/internal/service/publisher_rule_test.go`
13. `services/api-dashboard/internal/handler/publisher_app.go`
14. `services/api-dashboard/internal/handler/api_key.go`
15. `services/api-dashboard/internal/handler/publisher_rule.go`
16. `services/api-dashboard/internal/router/router.go` — add PublisherApp, APIKey, PublisherRule to Handlers struct, register routes
17. `services/api-dashboard/cmd/server/main.go` — wire new repos, services, handlers into DI

### Step 3: OpenAPI spec

18. `packages/proto/dashboard.yaml` — add all new paths and schemas

### Step 4: TypeScript (parallel once Go is done and openapi-typescript regenerated)

19. Regenerate TS client: `pnpm openapi-typescript`
20. `apps/dashboard/app/(dashboard)/apps/page.tsx`
21. `apps/dashboard/app/(dashboard)/apps/[id]/page.tsx`
22. `apps/dashboard/components/publisher/PublisherAppsList.tsx`
23. `apps/dashboard/components/publisher/CreateAppDialog.tsx`
24. `apps/dashboard/components/publisher/AppOverviewTab.tsx`
25. `apps/dashboard/components/publisher/AppApiKeysTab.tsx`
26. `apps/dashboard/components/publisher/ApiKeyRevealModal.tsx`
27. `apps/dashboard/components/publisher/AppRulesTab.tsx`
28. `apps/dashboard/components/publisher/RuleEditorDialog.tsx`
29. `apps/dashboard/hooks/usePublisherApps.ts`
30. `apps/dashboard/hooks/usePublisherApp.ts`
31. `apps/dashboard/hooks/useCreatePublisherApp.ts`
32. `apps/dashboard/hooks/useUpdatePublisherApp.ts`
33. `apps/dashboard/hooks/useApiKeys.ts`
34. `apps/dashboard/hooks/useCreateApiKey.ts`
35. `apps/dashboard/hooks/useRevokeApiKey.ts`
36. `apps/dashboard/hooks/usePublisherRules.ts`
37. `apps/dashboard/hooks/useCreateRule.ts`
38. `apps/dashboard/hooks/useUpdateRule.ts`
39. `apps/dashboard/hooks/useDeleteRule.ts`
40. Update sidebar nav for publisher org type

## 15. Risks & Open Questions

### Risks

- **sqlc query for total count on list endpoints**: sqlc does not natively generate COUNT queries alongside list queries. Each list endpoint requires a separate `CountPublisherAppsByOrg` query alongside `ListPublisherAppsByOrg`. This doubles query count per list call. Mitigation: accept the 2-query pattern for now; add CTEs in Phase 5 if latency becomes an issue.
- **JSONB config validation in Go**: config shape validation must be in the service layer (not DB CHECK constraint). Each rule type needs explicit struct deserialization and field checks. Risk of incomplete validation leaving malformed configs. Mitigation: define explicit Go structs per rule type and json.Unmarshal into them for validation.
- **One-time key display in UI**: the full key must survive the POST response through TanStack Query cache into the modal. If the user closes the modal and the query cache is invalidated (or the page reloads), the key is gone. This is correct behavior per FR-2, but UX risk if the user did not copy. Mitigation: keep "I have copied this key" as a required acknowledgment before closing; add explicit warning text.

### Open Questions

None — all decisions resolved per the spec inputs. The following were decided upfront and carried forward:
- SHA-256 (not argon2) for API key hashing
- key_prefix = first 8 chars of plaintext
- Pagination: offset-based, default 20, max 100
- 5 rule types with JSONB config
- Hard delete for rules, soft deactivation (is_active=false) for apps
- No endpoint for hard-deleting publisher apps in Phase 2
