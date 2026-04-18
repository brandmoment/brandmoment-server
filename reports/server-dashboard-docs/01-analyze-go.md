# Go Diagnostics: api-dashboard Public API Surface & Architecture

**Agent:** go-diagnostics
**Date:** 2026-04-18
**Scope:** `services/api-dashboard/` — full read-only extraction for documentation purposes

---

## 1. Architecture Overview

### Service Structure

The service follows the layered architecture defined in `.claude/rules/go-backend.md`:

```
services/api-dashboard/
├── cmd/server/main.go           # Entry point: config load, DI wiring, OTel init, HTTP server + graceful shutdown
├── internal/
│   ├── config/config.go         # Env-var config (manual os.Getenv, no envconfig library)
│   ├── router/router.go         # chi router, middleware chain, route registration
│   ├── handler/
│   │   ├── health.go            # GET /healthz
│   │   └── organization.go      # POST/GET /api/v1/organizations, GET /api/v1/organizations/{id}
│   ├── httputil/response.go     # Shared response helpers: RespondJSON / RespondError
│   ├── middleware/auth.go       # JWT validation + org membership check + RequireRole
│   ├── model/organization.go    # Organization domain type + sentinel errors
│   ├── repository/organization.go  # Wraps sqlc-generated Queries
│   └── service/
│       ├── organization.go      # Business logic + OTel spans + slog
│       └── organization_test.go # Table-driven unit tests, mock repo
├── go.mod
└── go.sum
```

### Dependency Injection Pattern

Wiring happens exclusively in `main.go` — no global state, no `init()`. Construction chain:

```
pgxpool.Pool
  └─► repository.NewOrganizationRepository(pool)
        └─► service.NewOrganizationService(repo, tp)
              └─► handler.NewOrganizationHandler(svc)

sdktrace.TracerProvider (tp)
  └─► middleware.NewAuth(cfg.JWTSecret)

router.NewRouter(&Handlers{Health, Organization}, auth)
  └─► http.Server{Addr: ":"+cfg.Port, Handler: mux}
```

All dependencies are constructor-injected. The `OrganizationRepository` interface is defined in the `repository` package, allowing the `service` layer to depend on it without knowing the concrete type — correct for testability.

---

## 2. Existing Endpoints

### Route Table

| Method | Path                          | Handler                              | Auth Middleware                              | RBAC                                      |
|--------|-------------------------------|--------------------------------------|----------------------------------------------|-------------------------------------------|
| GET    | `/healthz`                    | `HealthHandler.Check`                | None                                         | None                                      |
| POST   | `/api/v1/organizations`       | `OrganizationHandler.Create`         | `ValidateJWT` + `X-Org-ID` membership check  | `RequireRole("viewer","editor","admin","owner")` |
| GET    | `/api/v1/organizations`       | `OrganizationHandler.List`           | `ValidateJWT` + `X-Org-ID` membership check  | `RequireRole("viewer","editor","admin","owner")` |
| GET    | `/api/v1/organizations/{id}`  | `OrganizationHandler.GetByID`        | `ValidateJWT` + `X-Org-ID` membership check  | `RequireRole("viewer","editor","admin","owner")` |

### Handler Behavior Details

**GET /healthz**
- Response: `{"data": {"status": "ok"}}` — HTTP 200
- No authentication required

**POST /api/v1/organizations**
- Request body: `{"type": "admin|publisher|brand", "name": "string", "slug": "string"}`
- Validates: all three fields required; `type` must be one of `admin`, `publisher`, `brand`
- Response: `{"data": <Organization>}` — HTTP 201
- On error: `{"error": {"code": "INVALID_BODY"|"INVALID_INPUT", "message": "..."}}` — HTTP 400

**GET /api/v1/organizations**
- No query parameters
- Returns all organizations the authenticated user is a member of (IDs sourced from JWT `orgs[]` claim)
- Response: `{"data": [<Organization>, ...]}` — HTTP 200

**GET /api/v1/organizations/{id}**
- Path param: `id` (UUID)
- Additional access check in handler: verifies `id` is in the user's `orgIDs` list from context (JWT-derived), before querying DB
- Response: `{"data": <Organization>}` — HTTP 200
- On not found: `{"error": {"code": "NOT_FOUND", "message": "..."}}` — HTTP 404

---

## 3. Auth Flow

### JWT Validation (`ValidateJWT` middleware)

File: `/services/api-dashboard/internal/middleware/auth.go`

Step-by-step flow:

1. Extract `Authorization: Bearer <token>` header. Missing/malformed → 401 `UNAUTHORIZED`.
2. Parse and validate JWT using `golang-jwt/jwt/v5` with HMAC shared secret (`cfg.JWTSecret`).
   - **Note:** Current implementation uses a symmetric HMAC secret, NOT asymmetric JWKS validation against BetterAuth. The spec (§11) and CLAUDE.md specify JWKS validation — this is a deviation from spec.
3. Extract `X-Org-ID` header. Missing → 400 `MISSING_ORG_ID`.
4. Parse `X-Org-ID` as UUID. Invalid format → 400 `INVALID_ORG_ID`.
5. Scan JWT `orgs[]` claim: collect all `org_id` values into `orgIDs` slice; if `X-Org-ID` matches an entry, capture `role`.
6. If `role` is empty (user not a member of `X-Org-ID` org) → 403 `FORBIDDEN`.
7. Store `org_id` (active org), `role`, and `org_ids` (all user's orgs) in request context via typed context keys.

### RBAC (`RequireRole` middleware)

Reads `role` from context (set by `ValidateJWT`). Compares against allowed roles list. If no match → 403 `FORBIDDEN`.

Current registration: all organization routes allow `viewer|editor|admin|owner` — effectively all authenticated members.

### Context Accessors (exported from middleware package)

| Function | Returns | Used by |
|---|---|---|
| `OrgIDFromContext(ctx)` | `uuid.UUID` | Handler, Service |
| `RoleFromContext(ctx)` | `string` | RequireRole, Handler |
| `OrgIDsFromContext(ctx)` | `[]uuid.UUID` | OrganizationHandler.List, OrganizationHandler.GetByID |

---

## 4. Data Flow: Request → DB

### Full Call Chain (Organization GetByID)

```
HTTP GET /api/v1/organizations/{id}
  └─► chi router
        └─► otelchi.Middleware("api-dashboard")        [OTel trace propagation]
              └─► chimiddleware.RequestID               [X-Request-ID injection]
                    └─► chimiddleware.RealIP            [real IP extraction]
                          └─► chimiddleware.Recoverer   [panic recovery]
                                └─► auth.ValidateJWT    [JWT parse + org membership]
                                      └─► auth.RequireRole("viewer",...) [role check]
                                            └─► handler.OrganizationHandler.GetByID
                                                  │  uuid.Parse(chi.URLParam "id")
                                                  │  slices.Contains(orgIDs, id)  [membership guard]
                                                  └─► service.OrganizationService.GetByID(ctx, id)
                                                        │  tracer.Start("OrganizationService.GetByID")
                                                        │  slog.InfoContext(ctx, "getting organization")
                                                        └─► repository.organizationRepo.GetByID(ctx, id)
                                                              │  r.q.GetOrganizationByID(ctx, pgtype.UUID)
                                                              │  [sqlc-generated]
                                                              │  SELECT id,type,name,slug,created_at,updated_at
                                                              │  FROM organizations WHERE id = $1
                                                              │  pgx.ErrNoRows → model.ErrNotFound
                                                              └─► toOrganization(row) → *model.Organization
                                                  handler.handleServiceError(w, err) or
                                                  httputil.RespondJSON(w, 200, org)
```

### List Call Chain (Organization List)

```
GET /api/v1/organizations
  [same middleware chain]
    └─► handler.OrganizationHandler.List
          │  middleware.OrgIDsFromContext(ctx)  → []uuid.UUID from JWT
          └─► service.OrganizationService.ListByIDs(ctx, ids)
                └─► repository.organizationRepo.ListByIDs(ctx, []pgtype.UUID)
                      └─► r.q.ListOrganizationsByIDs(ctx, pgIDs)
                            SELECT ... FROM organizations WHERE id = ANY($1::uuid[]) ORDER BY created_at DESC
```

### Error Mapping

| Source | Error | HTTP Status | Code |
|---|---|---|---|
| JSON decode failure | any error | 400 | `INVALID_BODY` |
| UUID parse failure | any error | 400 | `INVALID_ID` |
| Service layer | `model.ErrInvalidInput` | 400 | `INVALID_INPUT` |
| Service layer | `model.ErrNotFound` | 404 | `NOT_FOUND` |
| Service layer | `model.ErrUnauthorized` | 401 | `UNAUTHORIZED` |
| Service layer | any other error | 500 | `INTERNAL_ERROR` |
| pgx.ErrNoRows | (repo) | mapped to `model.ErrNotFound` before reaching handler | — |

---

## 5. OTel Integration

### Tracer Initialization

File: `/services/api-dashboard/cmd/server/main.go`

- Exporter: `otlptracegrpc` — gRPC export to OTLP endpoint
- Service name: `api-dashboard` (via `semconv.ServiceNameKey`)
- Endpoint: `cfg.OTLPEndpoint` (env `OTEL_EXPORTER_OTLP_ENDPOINT`, default `localhost:4317`)
- Transport: insecure (no TLS) — suitable for dev via docker-compose otel-collector
- Batch mode: `sdktrace.WithBatcher(exporter)`

### Trace Propagation in HTTP

`otelchi.Middleware("api-dashboard")` is the first middleware applied — propagates W3C `traceparent` headers from incoming requests and starts the root span for each request.

### Service-Level Spans

Each service method creates a child span:

```go
ctx, span := s.tracer.Start(ctx, "OrganizationService.Create")
defer span.End()
// ...
span.RecordError(err)  // on error
```

Tracer name: `"brandmoment/api-dashboard"` (set in `NewOrganizationService`).

Span names defined:
- `OrganizationService.Create`
- `OrganizationService.GetByID`
- `OrganizationService.ListByIDs`

### Logging

Uses `log/slog` with `slog.InfoContext(ctx, ...)` — context-aware structured logging. Each service operation logs before the repo call with relevant field attributes (`slog.String`, `slog.Int`). No metrics instrumentation beyond traces is present in current code.

---

## 6. Configuration

File: `/services/api-dashboard/internal/config/config.go`

| Env Var | Required | Default | Description |
|---|---|---|---|
| `PORT` | No | `8080` | HTTP listen port |
| `DATABASE_URL` | Yes | — | PostgreSQL connection string (pgx DSN) |
| `JWT_SECRET` | Yes | — | HMAC secret for JWT validation |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | `localhost:4317` | gRPC OTLP exporter endpoint |

Config is loaded via `config.Load()` which returns an error if required vars are missing. No library (envconfig, viper, etc.) is used — raw `os.Getenv`.

HTTP server timeouts:
- `ReadTimeout`: 10s
- `WriteTimeout`: 10s
- `IdleTimeout`: 60s

Graceful shutdown: SIGINT/SIGTERM → `server.Shutdown()` with 10s context timeout.

---

## 7. Key Go Dependencies

From `/services/api-dashboard/go.mod` (Go 1.25.0 — note: spec says Go 1.23, module declares 1.25):

| Package | Version | Purpose |
|---|---|---|
| `github.com/go-chi/chi/v5` | v5.2.1 | HTTP router |
| `github.com/golang-jwt/jwt/v5` | v5.2.2 | JWT parsing and validation |
| `github.com/google/uuid` | v1.6.0 | UUID type |
| `github.com/jackc/pgx/v5` | v5.9.1 | PostgreSQL driver + connection pool |
| `github.com/riandyrn/otelchi` | v0.10.1 | OTel middleware for chi |
| `go.opentelemetry.io/otel` | v1.34.0 | OTel API |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` | v1.34.0 | gRPC OTLP trace exporter |
| `go.opentelemetry.io/otel/sdk` | v1.34.0 | OTel SDK (TracerProvider, Batcher) |
| `go.opentelemetry.io/otel/trace` | v1.34.0 | OTel trace API |

Shared dependency (via `replace` directive pointing to `../../packages/shared-domain`):
- `github.com/brandmoment/brandmoment-server/packages/shared-domain` — sqlc-generated `db.Queries`, `db.Organization` model

### shared-domain package

sqlc version: v1.30.0. Config: `/packages/shared-domain/sqlc.yaml`
- Engine: PostgreSQL
- SQL package: `pgx/v5`
- Emits: JSON tags, empty slices
- Schema source: `../../infra/migrations/`
- Query source: `queries/`

---

## 8. Database Schema (Current)

### Migration: `000001_create_organizations`

Only one migration exists. The `organizations` table:

```sql
CREATE TABLE IF NOT EXISTS organizations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type       TEXT NOT NULL CHECK (type IN ('admin', 'publisher', 'brand')),
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_organizations_slug ON organizations (slug);
```

### sqlc Queries Implemented

| Query name | SQL operation | Used by |
|---|---|---|
| `GetOrganizationByID` | `SELECT ... WHERE id = $1` | `repo.GetByID` |
| `ListOrganizationsByIDs` | `SELECT ... WHERE id = ANY($1::uuid[])` | `repo.ListByIDs` |
| `InsertOrganization` | `INSERT ... RETURNING *` | `repo.Insert` |

---

## 9. Response Envelope

All responses use the shared envelope from `/services/api-dashboard/internal/httputil/response.go`:

```json
// Success
{"data": <payload>}

// Error
{"error": {"code": "ERROR_CODE", "message": "human-readable message"}}
```

The `Response` struct uses `omitempty` — so `data` is absent on errors and `error` is absent on success. This matches the spec envelope format from §9: `{ data, error: null } | { data: null, error: { code, message } }` — with minor difference: the `null` fields are omitted rather than explicit `null`.

---

## 10. Test Coverage

File: `/services/api-dashboard/internal/service/organization_test.go`

Table-driven unit tests for all three service methods:

| Test | Cases |
|---|---|
| `TestOrganizationService_Create` | valid org; empty name; empty slug; empty type; invalid type |
| `TestOrganizationService_GetByID` | found; not found |
| `TestOrganizationService_ListByIDs` | returns organizations (2 items); empty list |

Mock pattern: `mockOrgRepo` struct with function fields (`insertFn`, `getByIDFn`, `listFn`) implements the `OrganizationRepository` interface inline per test case. Uses `noop.NewTracerProvider()` — no OTel side effects in tests.

No handler-level or integration tests exist.

---

## 11. What's Missing vs Spec (Sections 7–9)

### 11.1 Data Models — Spec §7 vs Code

The spec defines 9 tables across 4 domains. Currently only 1 is implemented.

**Identity / Tenancy (§7.1)**

| Table | Spec Fields | Status |
|---|---|---|
| `organization` | id, type, name, slug, created_at | Implemented (migration 000001) |
| `user` | id, email, name, created_at | Missing — no migration, no model, no handler |
| `org_membership` | id, org_id, user_id, role | Missing — no migration, no model, no handler |

**Publisher Domain (§7.2)**

| Table | Spec Fields | Status |
|---|---|---|
| `publisher_app` | id, org_id, name, bundle_id, platform, created_at | Missing |
| `api_key` | id, publisher_app_id, key_hash, key_prefix, name, last_used_at, revoked_at | Missing |
| `publisher_rule` | id, org_id, scope, app_id?, category_blocklist, brand_blocklist, frequency_cap_per_day, geo_allowlist, platform_allowlist | Missing |

**Brand / Campaign Domain (§7.3)**

| Table | Spec Fields | Status |
|---|---|---|
| `campaign` | id, org_id, name, status, targeting jsonb, budget_placeholder, created_at | Missing |
| `creative` | id, campaign_id, type, storage_key, entry_html_path, dimensions, checksum, validated_at | Missing |

**Matching Decisions (§7.4)**

| Table | Spec Fields | Status |
|---|---|---|
| `matching_decision` | id, session_id, publisher_app_id, campaign_id?, decision, reasons jsonb, created_at | Missing (SDK API concern, but table needed for dashboard read) |

### 11.2 API Endpoints — Spec §9 vs Code

The spec lists 14 endpoint groups. Currently only organizations partial CRUD is implemented, and path convention differs from spec.

| Spec Path | Spec Method | Status | Notes |
|---|---|---|---|
| `GET /v1/me` | GET | Missing | User profile endpoint |
| `POST /v1/auth/session/exchange` | POST | Missing | BetterAuth JWT → server JWT (spec marks this as optional) |
| `GET /v1/orgs/:id` | GET | Partial | Implemented as `GET /api/v1/organizations/{id}` — path prefix differs (`/api/v1` vs `/v1`), resource name differs (`organizations` vs `orgs`) |
| `POST /v1/orgs/:id/invites` | POST | Missing | Org member invitations |
| `GET /v1/publisher-apps` | GET | Missing | Publisher app listing |
| `POST /v1/publisher-apps` | POST | Missing | Publisher app creation |
| `POST /v1/publisher-apps/:id/api-keys` | POST | Missing | API key provisioning (argon2 hash + plaintext return) |
| `DELETE /v1/api-keys/:id` | DELETE | Missing | API key revocation |
| `GET /v1/publisher-rules` | GET | Missing | Publisher rules listing |
| `PUT /v1/publisher-rules/:id` | PUT | Missing | Publisher rules update |
| `GET /v1/campaigns` | GET | Missing | Campaign listing |
| `POST /v1/campaigns` | POST | Missing | Campaign creation |
| `POST /v1/campaigns/:id/creatives` | POST | Missing | HTML5 creative upload (multipart → S3) |
| `PATCH /v1/campaigns/:id/status` | PATCH | Missing | Campaign status transitions |
| `GET /v1/embeds/rill/signed-url` | GET | Missing | Rill embed URL generation |

### 11.3 Path Prefix Discrepancy

Spec §9 uses `/v1/...` paths. Implementation uses `/api/v1/...`. This needs to be reconciled in the OpenAPI spec (`packages/proto/dashboard.yaml`) — which the spec says is the source of truth but does not yet exist in the repo.

### 11.4 JWT Validation Approach

Spec §11 and CLAUDE.md state: use `golang-jwt/jwt/v5` to validate against **BetterAuth JWKS endpoint** (asymmetric). Current code uses HMAC symmetric secret (`cfg.JWTSecret`). This means the current auth is not compatible with BetterAuth token issuance. A JWKS-based `Keyfunc` is required.

### 11.5 Missing: OpenAPI Codegen Integration

Spec §5 states `packages/proto/` contains OpenAPI 3.1 spec as source of truth, with:
- `oapi-codegen` → Go server stubs
- `openapi-typescript` → TypeScript client

Currently `packages/proto/` directory structure is not reflected in the code — handler types are handwritten, not generated from OpenAPI spec. No `dashboard.yaml` spec file found in repo.

### 11.6 Missing: Mutations on Organizations (Update/Delete)

The Organizations resource only has Create, GetByID, List. No Update or Delete endpoints exist, and these are not mentioned explicitly in the spec either — but standard CRUD would require them eventually.

### 11.7 Missing: Pagination

`GET /api/v1/organizations` returns all user orgs with no pagination parameters. The spec does not define pagination explicitly for orgs, but for `publisher_apps` and `campaigns` (high-volume lists) pagination will be needed.

---

## 12. Structural Notes for Documentation

The following items are architecture decisions worth documenting explicitly:

1. **Organizations are not sub-resources** — they have no `org_id` column. Access control is purely JWT-membership based (user sees only orgs in their `orgs[]` claim). This is the correct multi-tenancy model per CLAUDE.md §Multi-Tenancy.

2. **Active org selection via `X-Org-ID` header** — the client selects which org context is active per request. The middleware validates this against the JWT's `orgs[]` list. This pattern handles multi-org users correctly.

3. **`OrgIDsFromContext` vs `OrgIDFromContext`** — two distinct context values: the active org (`ctxOrgID`) and all user's orgs (`ctxOrgIDs`). The List and GetByID handlers for organizations use `ctxOrgIDs` (all memberships), not just the active org — correct for a "list my orgs" pattern.

4. **Repository interface defined in repository package** — the service depends on `repository.OrganizationRepository` (interface), not a concrete type. This is testable but violates the stricter interpretation of dependency inversion (interface should be in the consumer's package). Minor point for a team this size.

5. **No metrics** — only traces (spans) are wired. No counters, histograms, or gauges. Spec §12 requires request rate/latency/error per endpoint, DB pool metrics.
