# Phase 0 (Dashboard Foundation) + Phase 1 (Backend Identity + Dashboard Auth Pages)

## 1. Context & Problem

### Current System State

**Backend (`services/api-dashboard/`):**
- Organizations CRUD: `POST /api/v1/organizations`, `GET /api/v1/organizations`, `GET /api/v1/organizations/{id}` — implemented
- JWT middleware: HMAC symmetric validation via `JWT_SECRET` env var — implemented but marked for replacement
- OTel tracing via OTLP gRPC — implemented
- Chi router with middleware chain (OTel → RequestID → RealIP → Recoverer → ValidateJWT → RequireRole) — implemented
- 1 migration: `000001_create_organizations` — users and org_memberships tables are absent
- No OpenAPI spec exists; path prefix is `/api/v1` (spec will use `/v1` — reconciliation required)
- Config: `Config{Port, DatabaseURL, JWTSecret, OTLPEndpoint}` — no JWKS URL field yet

**Dashboard (`apps/dashboard/`):**
- Completely empty — no files

**Shared (`packages/shared-domain/`):**
- sqlc config at `packages/shared-domain/sqlc.yaml` — reads schema from `infra/migrations/`
- No query files for users or org_memberships

### Problem / User Need

The platform cannot authenticate real users yet:
1. The backend issues no tokens — BetterAuth (dashboard) is not set up
2. The HMAC JWT validation must be replaced with JWKS asymmetric validation (BetterAuth issues RS256/ES256 tokens)
3. There is no `users` or `org_memberships` table, making identity-based queries impossible
4. `GET /v1/me` (current user profile) does not exist
5. Org invites endpoint does not exist
6. The dashboard has no UI at all

### Business Goals

- Enable user registration and login via BetterAuth self-hosted in Next.js
- Establish the OpenAPI contract as single source of truth for Go stubs and TS client types
- Deliver auth pages (login, signup, invite acceptance, onboarding) to unblock Publisher Phase 2 work
- Support multi-org users with an org switcher from day one

---

## 2. Goals & Non-Goals

### Goals

- Define `packages/proto/dashboard.yaml` as OpenAPI 3.1 source of truth
- Set up oapi-codegen (Go server stubs) and openapi-typescript (TS client) with Makefile targets
- Migrate JWT validation from HMAC symmetric to BetterAuth JWKS asymmetric (RS256)
- Add `users` and `org_memberships` migrations, sqlc queries, model, repo, service, handler
- Implement `GET /v1/me` and `POST /v1/orgs/{id}/invites` endpoints
- Reconcile path prefix: move existing `/api/v1` routes to `/v1`
- Scaffold Next.js 15 app with BetterAuth, shadcn/ui, Tailwind v4, TanStack Query
- Implement auth pages: `/login`, `/signup`, `/accept-invite/:token`, `/onboarding`, org switcher

### Non-Goals

- Publisher apps, API keys, rules — Phase 2
- Campaigns, creatives — Phase 3
- Analytics endpoints and Rill embed — Phase 4
- OTel metrics and DB pool metrics — Phase 5
- Mobile SDK (`services/api-sdk/`) — v2
- Email delivery for invites (stub endpoint only; actual email sending is out of scope)
- Social OAuth providers beyond BetterAuth defaults (configurable but not tested in this phase)

---

## 3. User Stories

- As a new user I want to register with email/password so that I can access the platform.
- As a registered user I want to log in so that I receive an authenticated session.
- As an authenticated user I want to complete onboarding (choose org type, name it, create first resource stub) so that I am ready to use publisher or brand features.
- As a multi-org user I want to switch the active org from the topbar so that I can operate on behalf of different organizations.
- As an org owner I want to invite a collaborator by email so that they can join my organization.
- As an invitee I want to accept an invite via a link so that I am added to the org.
- As an authenticated user hitting any Go API endpoint I want my BetterAuth-issued JWT to be validated correctly so that I am not rejected due to signature algorithm mismatch.
- As a developer I want a single OpenAPI spec so that Go server stubs and TS client types are always in sync with the contract.

---

## 4. Scope

### In Scope

**SQL layer:**
- Migration `000002_create_users`
- Migration `000003_create_org_memberships`
- sqlc query files for both entities

**Go backend (new):**
- OpenAPI spec `packages/proto/dashboard.yaml`
- oapi-codegen config + Makefile target
- Config: add `BetterAuthJWKSURL` field; make `JWTSecret` optional (used only in dev HMAC mode)
- JWKS JWT validation replacing HMAC in `internal/middleware/auth.go`
- Route prefix change: `/api/v1` → `/v1`
- `internal/model/user.go`
- `internal/repository/user.go`
- `internal/service/user.go`
- `internal/handler/user.go` (`GET /v1/me`)
- `internal/handler/org_invite.go` (`POST /v1/orgs/{id}/invites`)
- `internal/service/user_test.go` (table-driven)
- Router updates: register new routes, add User handler to `Handlers` struct

**TypeScript (new app scaffold):**
- `apps/dashboard/` Next.js 15 scaffold
- BetterAuth server setup (`apps/dashboard/lib/auth.ts`)
- BetterAuth client (`apps/dashboard/lib/auth-client.ts`)
- Next.js API route handler (`apps/dashboard/app/api/auth/[...all]/route.ts`)
- openapi-typescript setup + codegen script
- Generated TS client (`apps/dashboard/lib/api-client.ts`)
- TanStack Query provider (`apps/dashboard/app/providers.tsx`)
- Root layout with sidebar + topbar shell (`apps/dashboard/app/layout.tsx`)
- Sidebar component (`apps/dashboard/components/Sidebar.tsx`)
- Topbar component with org switcher (`apps/dashboard/components/Topbar.tsx`)
- OrgSwitcher component (`apps/dashboard/components/OrgSwitcher.tsx`)
- Auth pages: `/login`, `/signup`, `/accept-invite/[token]`, `/onboarding`
- Middleware for auth-guarded routes (`apps/dashboard/middleware.ts`)

### Out of Scope

- Publisher and brand page routes beyond onboarding stub
- Rill proxy route
- Real email delivery for invites
- Social OAuth provider credentials (BetterAuth configured but providers require env keys)

---

## 5. Functional Requirements

- FR-1: `packages/proto/dashboard.yaml` defines all v1 endpoints using OpenAPI 3.1; it is the single source of truth for both Go stubs and TS types.
- FR-2: Running `make codegen` regenerates Go server stubs from the OpenAPI spec (oapi-codegen) and TS types (openapi-typescript) without manual changes.
- FR-3: Go JWT middleware validates `Authorization: Bearer <token>` using BetterAuth JWKS endpoint (asymmetric RS256); HMAC path is removed from production code.
- FR-4: `GET /v1/me` returns the authenticated user's profile and their org memberships extracted from the validated JWT claims.
- FR-5: `POST /v1/orgs/{id}/invites` accepts `{email, role}`, validates org membership of the caller (must be `admin` or `owner`), persists the invite record, and returns `{invite_id, token}`.
- FR-6: All existing organization routes are reachable at `/v1/organizations` (prefix changed from `/api/v1`); no breaking change in payload shape.
- FR-7: `users` table is seeded on first BetterAuth login via a BetterAuth database adapter hook or via a `GET /v1/me` upsert — see assumption in Section 14.
- FR-8: `org_memberships` table stores the canonical role for each user–org pair; JWT claims are derived from this table by BetterAuth at token issuance.
- FR-9: Dashboard `/login` page authenticates via BetterAuth `signIn.email` client method; on success redirects to `/` (or `/onboarding` if no org memberships).
- FR-10: Dashboard `/signup` page calls BetterAuth `signUp.email`, then redirects to `/onboarding`.
- FR-11: Dashboard `/onboarding` wizard: step 1 — choose org type (publisher/brand); step 2 — org name + slug input, calls `POST /v1/organizations`; step 3 — confirmation with link to first resource page.
- FR-12: Dashboard `/accept-invite/:token` validates the token via `POST /v1/orgs/{id}/invites/accept` (Phase 2 endpoint — stub page only in this phase, shows "invite acceptance coming soon" with the token displayed).
- FR-13: Org switcher in topbar renders all orgs from session, updates `activeOrgId` in client state, and sets `X-Org-ID` header on all subsequent API calls.
- FR-14: Unauthenticated users accessing any route except `/login`, `/signup`, `/accept-invite/*` are redirected to `/login` via Next.js middleware.

---

## 6. API Changes

### Existing Endpoints Affected

| Change | Before | After |
|--------|--------|-------|
| Path prefix | `/api/v1/organizations` | `/v1/organizations` |
| JWT algorithm | HMAC HS256 (`JWT_SECRET`) | RS256 via JWKS (`BETTERAUTH_JWKS_URL`) |

No request/response shape changes to organization endpoints.

### New Endpoints

#### GET /v1/me

**Auth:** ValidateJWT (no RequireRole; any authenticated user)

**Request:** No body. `Authorization: Bearer <token>` + `X-Org-ID: <uuid>` headers.

**Response 200:**
```json
{
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "Alice",
    "created_at": "2026-04-18T00:00:00Z",
    "orgs": [
      {"org_id": "uuid", "role": "owner"}
    ]
  }
}
```

**Errors:**
- `401 UNAUTHORIZED` — invalid or missing JWT
- `404 NOT_FOUND` — user record not found in `users` table (should not happen post-upsert; defensive)

**Notes:** The user's org list is read from JWT claims (`orgs[]`), not from a DB query, to keep this endpoint fast. The `id`, `email`, `name` fields are fetched from the `users` table by `sub` claim (UUID).

#### POST /v1/orgs/{id}/invites

**Auth:** ValidateJWT + RequireRole("admin", "owner")

**Request:**
```json
{
  "email": "invitee@example.com",
  "role": "editor"
}
```

**Response 201:**
```json
{
  "data": {
    "invite_id": "uuid",
    "token": "opaque-random-token",
    "email": "invitee@example.com",
    "role": "editor",
    "org_id": "uuid",
    "expires_at": "2026-04-25T00:00:00Z"
  }
}
```

**Errors:**
- `400 INVALID_INPUT` — missing email, invalid role value
- `400 INVALID_ID` — org ID in path is not a valid UUID
- `401 UNAUTHORIZED` — invalid JWT
- `403 FORBIDDEN` — caller role is viewer or editor
- `404 NOT_FOUND` — org not found or caller is not a member

**Implementation note:** `token` is generated as `crypto/rand` 32-byte value, base64url-encoded. The `org_invites` table is required (see Data Model). Email delivery is out of scope; the token is returned in the response only.

---

## 7. Data Model

### New Tables

#### `000002_create_users.up.sql`

```sql
CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email ON users (email);
```

```sql
-- 000002_create_users.down.sql
DROP TABLE IF EXISTS users;
```

**Notes:**
- No `password_hash` column — credentials are managed entirely by BetterAuth (its own internal tables). This `users` table is the platform's domain record, created/upserted on first BetterAuth login.
- No `updated_at` — user profile mutations are not in Phase 1 scope.

#### `000003_create_org_memberships.up.sql`

```sql
CREATE TABLE IF NOT EXISTS org_memberships (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'editor', 'viewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, user_id)
);

CREATE INDEX idx_org_memberships_org_id ON org_memberships (org_id);
CREATE INDEX idx_org_memberships_user_id ON org_memberships (user_id);
```

```sql
-- 000003_create_org_memberships.down.sql
DROP TABLE IF EXISTS org_memberships;
```

#### `000004_create_org_invites.up.sql`

Required by `POST /v1/orgs/{id}/invites`.

```sql
CREATE TABLE IF NOT EXISTS org_invites (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,
    role       TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'editor', 'viewer')),
    token      TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_org_invites_token ON org_invites (token);
CREATE INDEX idx_org_invites_org_id ON org_invites (org_id);
```

```sql
-- 000004_create_org_invites.down.sql
DROP TABLE IF EXISTS org_invites;
```

### sqlc Query Files

#### `packages/shared-domain/queries/users.sql`

```sql
-- name: GetUserByID :one
SELECT id, email, name, created_at
FROM users
WHERE id = @id;

-- name: GetUserByEmail :one
SELECT id, email, name, created_at
FROM users
WHERE email = @email;

-- name: UpsertUser :one
INSERT INTO users (id, email, name, created_at)
VALUES (@id, @email, @name, @created_at)
ON CONFLICT (email) DO UPDATE
  SET name = EXCLUDED.name
RETURNING *;
```

#### `packages/shared-domain/queries/org_memberships.sql`

```sql
-- name: GetMembershipByUserAndOrg :one
SELECT id, org_id, user_id, role, created_at
FROM org_memberships
WHERE user_id = @user_id AND org_id = @org_id;

-- name: ListMembershipsByUser :many
SELECT id, org_id, user_id, role, created_at
FROM org_memberships
WHERE user_id = @user_id
ORDER BY created_at ASC;

-- name: InsertMembership :one
INSERT INTO org_memberships (id, org_id, user_id, role, created_at)
VALUES (@id, @org_id, @user_id, @role, @created_at)
RETURNING *;
```

#### `packages/shared-domain/queries/org_invites.sql`

```sql
-- name: InsertOrgInvite :one
INSERT INTO org_invites (id, org_id, email, role, token, expires_at, created_at)
VALUES (@id, @org_id, @email, @role, @token, @expires_at, @created_at)
RETURNING *;

-- name: GetOrgInviteByToken :one
SELECT id, org_id, email, role, token, expires_at, accepted_at, created_at
FROM org_invites
WHERE token = @token;
```

### Migration Plan

Sequential execution:
1. `000001_create_organizations` — already applied
2. `000002_create_users` — new; no FK dependencies
3. `000003_create_org_memberships` — new; depends on `organizations` + `users`
4. `000004_create_org_invites` — new; depends on `organizations`

After writing migration files: `sqlc generate` from `packages/shared-domain/`.

### Multi-Tenancy

- `users` — top-level resource, no `org_id` column. Access via JWT sub claim.
- `org_memberships` — sub-resource; always filter by `user_id` (and optionally `org_id`).
- `org_invites` — sub-resource; always filter by `org_id = $1` in queries.

---

## 8. UI Changes

### New App: `apps/dashboard/`

Scaffold via `create-next-app` with App Router, TypeScript, Tailwind v4.

#### Directory Structure

```
apps/dashboard/
├── app/
│   ├── layout.tsx                    # Root layout: font, providers, metadata
│   ├── providers.tsx                 # TanStack Query + BetterAuth session provider
│   ├── (auth)/
│   │   ├── login/page.tsx
│   │   ├── signup/page.tsx
│   │   ├── accept-invite/[token]/page.tsx
│   │   └── onboarding/page.tsx
│   ├── (dashboard)/
│   │   ├── layout.tsx                # Dashboard shell: sidebar + topbar
│   │   └── page.tsx                  # Default redirect to /apps or /campaigns
│   └── api/
│       └── auth/
│           └── [...all]/route.ts     # BetterAuth catch-all handler
├── components/
│   ├── Sidebar.tsx
│   ├── Topbar.tsx
│   └── OrgSwitcher.tsx
├── lib/
│   ├── auth.ts                       # BetterAuth server instance
│   ├── auth-client.ts                # BetterAuth client instance
│   └── api-client.ts                 # openapi-fetch typed client
├── middleware.ts                     # Next.js edge middleware (auth guard)
├── next.config.ts
├── package.json
├── tailwind.config.ts
└── tsconfig.json
```

#### Auth Pages

**`/login`:**
- Email input + password input + submit button
- "Sign up" link → `/signup`
- On success: redirect to `/` (middleware handles onboarding redirect)
- Error: toast via sonner on invalid credentials

**`/signup`:**
- Name input + email input + password input + confirm password
- Calls BetterAuth `signUp.email({ name, email, password })`
- On success: redirect to `/onboarding`

**`/accept-invite/[token]`:**
- Phase 1 stub: displays "Invite acceptance is being set up. Your token: {token}"
- Full flow (POST to accept endpoint) deferred to Phase 2 when Backend Phase 2 is ready

**`/onboarding`:**
Multi-step wizard (client component):
- Step 1: Org type selection — two cards: "Publisher" / "Brand"
- Step 2: Org name (text input) + slug (auto-generated, editable). Calls `POST /v1/organizations`
- Step 3: Success screen with CTA — "Go to Apps" (publisher) or "Go to Campaigns" (brand)
- Progress indicator (step dots)

#### Layout Shell

**Sidebar (`components/Sidebar.tsx`):**
- Reads active org type from session context
- Publisher nav: Apps, Rules, Analytics
- Brand nav: Campaigns, Analytics
- Admin nav: Organizations, All Campaigns, All Apps, Analytics
- Uses shadcn/ui `Button` (variant ghost) for nav items

**Topbar (`components/Topbar.tsx`):**
- Left: hamburger (mobile) + app logo
- Center: `OrgSwitcher`
- Right: user avatar menu (profile stub, logout)

**OrgSwitcher (`components/OrgSwitcher.tsx`):**
- Renders a shadcn/ui `DropdownMenu` listing all orgs from session
- Selecting an org updates `activeOrgId` in React state (context or Zustand — see assumption)
- `activeOrgId` is forwarded as `X-Org-ID` header on all API client calls

#### Navigation Changes

No existing navigation. New routes:

| Route | Auth Required | Notes |
|-------|--------------|-------|
| `/login` | No | Redirect to `/` if already authed |
| `/signup` | No | Redirect to `/` if already authed |
| `/accept-invite/[token]` | No | Stub in Phase 1 |
| `/onboarding` | Yes | Redirect to `/login` if not authed |
| `/` | Yes | Dashboard shell home (redirect placeholder) |

---

## 9. State & Flows

### Auth Flow (Happy Path)

```
User → /signup → BetterAuth signUp.email
  → BetterAuth creates auth record
  → Redirect to /onboarding
    → POST /v1/organizations (with JWT + X-Org-ID from new org)
      → org created, membership row inserted
        → Redirect to /apps (publisher) or /campaigns (brand)
```

### Login Flow (Happy Path)

```
User → /login → BetterAuth signIn.email
  → BetterAuth issues JWT with org memberships
  → Redirect to /
    → middleware checks session → passes
      → if no org memberships → redirect to /onboarding
      → else → dashboard home
```

### GET /v1/me Flow

```
Client → GET /v1/me
  → ValidateJWT: JWKS fetch (cached) → verify RS256 signature
  → extract sub (user UUID) from claims
  → UserService.GetMe(ctx, userID uuid.UUID) → UpsertUser (first-time login creates record)
  → return user fields + orgs[] from claims
```

### POST /v1/orgs/{id}/invites Flow

```
Client → POST /v1/orgs/{id}/invites {email, role}
  → ValidateJWT → RequireRole("admin", "owner")
  → OrgInviteService.Create(ctx, orgID, email, role)
    → validate role is in ('admin', 'editor', 'viewer') — owner cannot be invited
    → generate token: crypto/rand 32 bytes → base64url
    → InsertOrgInvite with expires_at = now() + 7 days
    → return invite record
```

### JWKS Validation Flow

```
Request → ValidateJWT
  → fetch JWKS from BETTERAUTH_JWKS_URL (HTTP GET, cached in-process with 1h TTL)
  → jwt.ParseWithClaims using keyfunc from jwks cache
  → verify alg = RS256, validate exp, iss if configured
  → extract sub, orgs[] from claims
  → validate X-Org-ID against orgs[]
```

### Edge Cases

- JWKS fetch fails on startup: log error, reject all requests with 503 until cache warms
- JWKS cache miss (TTL expired): re-fetch synchronously (acceptable p99 hit once per hour)
- User JWT contains org that no longer has a membership row: not validated at JWT layer; API queries will fail naturally with 403/404
- Onboarding: user clicks back to step 1 — org type selection resets; no partial org is created
- Invite token already accepted: `accepted_at` is not null — `GET /v1/orgs/{id}/invites/accept` will return 409 (Phase 2 endpoint)

### Error Handling

All Go handlers use existing `httputil.RespondError(w, status, code, message)`. No new error response shape.

New sentinel errors in `internal/model/user.go`:
- `ErrUserNotFound = errors.New("user not found")` (maps to `model.ErrNotFound` — reuse existing)
- Invite service reuses `model.ErrInvalidInput` for bad role values

---

## 10. Non-Functional Requirements

### Performance

- `GET /v1/me`: target p99 < 50ms (single SELECT by UUID PK + in-memory JWT claims)
- JWKS cache: in-process `sync.Map` or `sync.RWMutex`-guarded map; TTL 1 hour; no Redis dependency in this phase
- BetterAuth session fetch in Next.js Server Components: `auth()` is cached per request by Next.js

### Security

- HMAC `JWT_SECRET` env var removed from production config; field retained in `Config` struct for dev-mode compatibility only (see assumption in Section 14)
- JWKS endpoint URL must be HTTPS in production; no validation skipping for dev
- Invite tokens: `crypto/rand`, not `math/rand`; 32 bytes → 43-char base64url
- Invite token TTL: 7 days (hardcoded in service, extractable to config later)
- `POST /v1/orgs/{id}/invites`: role `owner` cannot be assigned via invite — service-level validation returns `ErrInvalidInput`
- BetterAuth cookie: `httpOnly`, `secure` (enforced by BetterAuth defaults in production)
- Next.js middleware runs at Edge: does not call Go API; validates BetterAuth session cookie only

### Multi-Tenancy

- `org_memberships.org_id` is always present; every membership query filters by `user_id` or `org_id`
- `GET /v1/me` does NOT filter by active org — returns all orgs from claims (cross-org endpoint by design)
- `POST /v1/orgs/{id}/invites` uses org ID from path, validated against caller's JWT membership — never trusts request body org_id

### Observability

- `UserService.GetMe` and `OrgInviteService.Create` each start an OTel child span: `UserService.GetMe`, `OrgInviteService.Create`
- `slog.InfoContext` on service entry with `user_id`, `org_id` attributes
- JWKS cache refresh logged at `slog.Info` level with `jwks_url`, `key_count`
- No new metrics in this phase

---

## 11. Dependencies

### Services Affected

- `services/api-dashboard/` — JWKS migration, new entities, new endpoints, route prefix change
- `apps/dashboard/` — new app (greenfield)
- `packages/shared-domain/` — new sqlc queries; `sqlc generate` must run after migrations

### External APIs / Libraries

**Go backend additions:**

| Package | Purpose |
|---------|---------|
| `MicahParks/keyfunc/v3` or `golang-jwt/jwx` | JWKS fetching + RS256 keyfunc for `golang-jwt/jwt/v5` |
| No other new packages | Existing chi, pgx, sqlc, OTel remain |

Assumption: use `github.com/MicahParks/keyfunc/v3` — integrates directly with `golang-jwt/jwt/v5` keyfunc interface.

**TypeScript additions (package.json for apps/dashboard):**

| Package | Purpose |
|---------|---------|
| `better-auth` | Auth server + client |
| `@tanstack/react-query` | Server state, data fetching |
| `openapi-fetch` | Type-safe HTTP client |
| `openapi-typescript` (devDep) | TS type generation from OpenAPI spec |
| `@openapitools/openapi-generator-cli` | Not used — using oapi-codegen for Go, openapi-typescript for TS |
| `sonner` | Toast notifications |
| `react-hook-form` | Form state management |
| `zod` | Schema validation |
| `@hookform/resolvers` | zod adapter for react-hook-form |

shadcn/ui components needed in Phase 1: `button`, `input`, `label`, `card`, `dropdown-menu`, `avatar`, `skeleton`, `toast` (via sonner).

### Infrastructure

- BetterAuth runs inside Next.js process — no new Docker service
- `BETTERAUTH_URL` env var: the public URL of the Next.js app (used by BetterAuth for callback URLs)
- `BETTERAUTH_SECRET` env var: secret for BetterAuth session signing
- `BETTERAUTH_JWKS_URL` env var: `{BETTERAUTH_URL}/api/auth/jwks` — added to Go backend config

### Makefile Targets (new)

```makefile
codegen:
    cd packages/proto && \
    oapi-codegen -config oapi-codegen.yaml dashboard.yaml > \
      ../../services/api-dashboard/internal/api/server.gen.go
    cd packages/proto && \
    npx openapi-typescript dashboard.yaml -o \
      ../../apps/dashboard/lib/api-types.gen.ts
```

---

## 12. Implementation Order

Per CLAUDE.md New Entity Checklist and project conventions: SQL first, then Go, then TypeScript. Within Go: model → repository → service → handler → router. SQL-builder must finish before Go-builder starts (sqlc-generated code is a Go dependency).

### Step-by-Step Order

#### SQL Layer (sql-builder)

1. `infra/migrations/000002_create_users.up.sql` + `.down.sql`
2. `infra/migrations/000003_create_org_memberships.up.sql` + `.down.sql`
3. `infra/migrations/000004_create_org_invites.up.sql` + `.down.sql`
4. `packages/shared-domain/queries/users.sql`
5. `packages/shared-domain/queries/org_memberships.sql`
6. `packages/shared-domain/queries/org_invites.sql`
7. Run `sqlc generate` — verify no errors

#### Go Layer (go-builder) — after sql-builder

8. `packages/proto/dashboard.yaml` — OpenAPI 3.1 spec
9. `packages/proto/oapi-codegen.yaml` — oapi-codegen config
10. Run `make codegen` → generates `services/api-dashboard/internal/api/server.gen.go`
11. `services/api-dashboard/internal/config/config.go` — add `BetterAuthJWKSURL string`; remove `JWTSecret` required validation (make optional)
12. `services/api-dashboard/internal/middleware/auth.go` — replace HMAC keyfunc with JWKS keyfunc (MicahParks/keyfunc); add JWKS cache initialization
13. `services/api-dashboard/internal/model/user.go` — `User` struct, `OrgMembership` struct
14. `services/api-dashboard/internal/repository/user.go` — interface + implementation wrapping sqlc
15. `services/api-dashboard/internal/service/user.go` — `GetMe` + `UpsertUser`; OTel span; slog
16. `services/api-dashboard/internal/handler/user.go` — `GET /v1/me`
17. `services/api-dashboard/internal/model/org_invite.go` — `OrgInvite` struct
18. `services/api-dashboard/internal/repository/org_invite.go`
19. `services/api-dashboard/internal/service/org_invite.go` — `Create`; token generation
20. `services/api-dashboard/internal/handler/org_invite.go` — `POST /v1/orgs/{id}/invites`
21. `services/api-dashboard/internal/router/router.go` — change prefix `/api/v1` → `/v1`; add User and OrgInvite handlers; update `Handlers` struct
22. `services/api-dashboard/cmd/server/main.go` — wire UserRepo, UserService, UserHandler, OrgInviteRepo, OrgInviteService, OrgInviteHandler; pass JWKS URL to Auth
23. `services/api-dashboard/internal/service/user_test.go` — table-driven tests
24. `services/api-dashboard/internal/service/org_invite_test.go` — table-driven tests
25. Run `go build ./...` + `go vet ./...` + `go test ./...`

#### TypeScript Layer (ts-builder) — can run in parallel with Go Layer after Step 7

26. `apps/dashboard/package.json` — dependencies
27. `apps/dashboard/next.config.ts`
28. `apps/dashboard/tailwind.config.ts`
29. `apps/dashboard/tsconfig.json`
30. `apps/dashboard/lib/auth.ts` — BetterAuth server: email/password plugin, database adapter (Postgres via `pg` or `@neondatabase/serverless`)
31. `apps/dashboard/lib/auth-client.ts` — `createAuthClient()`
32. `apps/dashboard/app/api/auth/[...all]/route.ts` — BetterAuth handler
33. Run openapi-typescript → `apps/dashboard/lib/api-types.gen.ts`
34. `apps/dashboard/lib/api-client.ts` — `createClient` from `openapi-fetch` using generated types
35. `apps/dashboard/app/providers.tsx` — `QueryClientProvider` + session context
36. `apps/dashboard/app/layout.tsx` — root layout
37. `apps/dashboard/middleware.ts` — auth guard
38. `apps/dashboard/components/Sidebar.tsx`
39. `apps/dashboard/components/OrgSwitcher.tsx`
40. `apps/dashboard/components/Topbar.tsx`
41. `apps/dashboard/app/(dashboard)/layout.tsx` — dashboard shell
42. `apps/dashboard/app/(auth)/login/page.tsx`
43. `apps/dashboard/app/(auth)/signup/page.tsx`
44. `apps/dashboard/app/(auth)/accept-invite/[token]/page.tsx`
45. `apps/dashboard/app/(auth)/onboarding/page.tsx`
46. Run `pnpm typecheck` + `pnpm lint`

---

## 13. Testing Strategy

### Unit Tests (Go)

Table-driven tests with mock interfaces (func fields, no testify/gomock). `noop.NewTracerProvider()` for OTel.

**`TestUserService_GetMe`:**

| Case | Input | Expected |
|------|-------|----------|
| user exists | valid userID UUID | returns `*User`, nil |
| user not found | unknown UUID | returns nil, `ErrNotFound` |
| db error | repo returns generic error | returns nil, wrapped error |

**`TestUserService_UpsertUser`:**

| Case | Input | Expected |
|------|-------|----------|
| new user | email not in DB | inserts, returns `*User` |
| existing user (same email) | email exists | updates name, returns `*User` |
| empty email | empty string | returns nil, `ErrInvalidInput` |

**`TestOrgInviteService_Create`:**

| Case | Input | Expected |
|------|-------|----------|
| valid invite | email, role=editor | returns `*OrgInvite` with non-empty token |
| role=owner | role="owner" | returns nil, `ErrInvalidInput` |
| invalid role | role="superadmin" | returns nil, `ErrInvalidInput` |
| db error | repo returns error | returns nil, wrapped error |

### Integration Tests

Not in scope for Phase 1. Infrastructure for table-driven service tests is sufficient.

### Manual Verification Steps

1. Apply migrations: `migrate -path infra/migrations -database "$DATABASE_URL" up` — verify no errors
2. Run `sqlc generate` — verify generated files in `packages/shared-domain/db/`
3. `go build ./...` — zero errors
4. `curl -X POST http://localhost:8080/v1/organizations` with valid BetterAuth JWT — verify route works at new prefix
5. `curl http://localhost:8080/api/v1/organizations` — verify 404 (old prefix removed)
6. `curl http://localhost:8080/v1/me` with valid JWT — verify user profile response
7. `curl -X POST http://localhost:8080/v1/orgs/{id}/invites` with owner JWT — verify invite token returned
8. Navigate to `http://localhost:3000/signup` — verify form renders, BetterAuth creates session
9. After signup, verify redirect to `/onboarding`
10. Complete onboarding — verify `POST /v1/organizations` is called, org created
11. Verify org switcher shows orgs from session
12. Log out, navigate to `/` — verify redirect to `/login`

---

## 14. Acceptance Criteria

- AC-1: `packages/proto/dashboard.yaml` exists and is valid OpenAPI 3.1; `make codegen` exits 0 and generates both `server.gen.go` and `api-types.gen.ts` without manual edits.
- AC-2: Go backend starts with `BETTERAUTH_JWKS_URL` set and no `JWT_SECRET`; `GET /healthz` returns 200.
- AC-3: A BetterAuth-issued RS256 JWT is accepted by `ValidateJWT`; a manually signed HMAC JWT is rejected with 401.
- AC-4: `GET /v1/me` with a valid BetterAuth JWT returns the user's email, name, and orgs array; response time p99 < 50ms (local dev).
- AC-5: `POST /v1/orgs/{id}/invites` with owner JWT and `{email, role: "editor"}` returns 201 with a non-empty `token` field.
- AC-6: `POST /v1/orgs/{id}/invites` with role `"owner"` returns 400 `INVALID_INPUT`.
- AC-7: `POST /v1/organizations` at `/v1/organizations` (new prefix) returns the same response as before; `/api/v1/organizations` returns 404.
- AC-8: Migrations 000002, 000003, 000004 apply cleanly and roll back cleanly.
- AC-9: `go build ./...` and `go vet ./...` and `go test ./...` all exit 0.
- AC-10: `apps/dashboard/` Next.js app starts (`pnpm dev`) without TypeScript errors.
- AC-11: `/login` page renders and accepts email/password; valid credentials create a BetterAuth session and redirect to `/`.
- AC-12: `/signup` page creates a new BetterAuth user and redirects to `/onboarding`.
- AC-13: `/onboarding` wizard completes all 3 steps and calls `POST /v1/organizations`; on success shows the confirmation step.
- AC-14: Unauthenticated navigation to `/onboarding` redirects to `/login`.
- AC-15: Org switcher renders all orgs from the session and updates `X-Org-ID` on API calls when switched.
- AC-16: `pnpm typecheck` exits 0 for `apps/dashboard/`.

---

## 15. Risks & Open Questions

### Risks

- **JWKS keyfunc library choice:** `MicahParks/keyfunc/v3` is the most commonly used adapter for `golang-jwt/jwt/v5`. If it lacks automatic background refresh, implement manual TTL-based refresh. Verify with `go get` before coding.
- **BetterAuth database adapter:** BetterAuth requires a database adapter to persist its own auth records (sessions, accounts). The Postgres adapter (`@better-auth/adapter-drizzle` or `@better-auth/adapter-pg`) must be configured. This creates BetterAuth-managed tables (separate from `users` in this spec) via BetterAuth's own migration. Verify which adapter is available and whether it conflicts with golang-migrate.
- **Route prefix breaking change:** Changing `/api/v1` → `/v1` breaks any client (tests, curl scripts, seed tools) that hardcodes the old prefix. Audit all usages before removing the old prefix.
- **UserID sync:** BetterAuth's internal user ID may differ from the UUID in the `users` table. The `sub` claim in the BetterAuth JWT must be a UUID matching `users.id`. Verify BetterAuth's `sub` claim format; if it is not a UUID, the `UpsertUser` logic must map it.

### Open Questions

1. **HMAC dev mode:** Should `JWT_SECRET` remain as a fallback for local dev (e.g., when BetterAuth is not running)? If yes, `ValidateJWT` needs a mode switch (env-based). If no, dev must always run BetterAuth. — Recommendation: keep dev-mode HMAC behind a `DEV_MODE=true` env flag, disabled in production.
2. **BetterAuth `sub` claim format:** Is the JWT `sub` field a UUID? If BetterAuth uses a different ID format (e.g., `user_<nanoid>`), the `users.id` column type must change to TEXT. Verify before writing the migration.
3. **Org switcher state management:** Is a React context sufficient for `activeOrgId`, or should Zustand be introduced? The spec defers this decision to ts-builder; a context is sufficient for Phase 1 given the small component tree.
4. **BetterAuth social OAuth:** Are Google/GitHub OAuth provider credentials available for dev? The spec configures email/password only; social providers are configured but not tested. Confirm which providers are in scope.
5. **`/accept-invite/:token` full implementation timing:** The spec stubs this page. The full backend accept endpoint (`POST /v1/orgs/{id}/invites/accept`) is not defined here. Should it move into Phase 1 or stay in Phase 2? Affects whether the invite flow is end-to-end testable in this phase.
