# Go Backend Scan — Phase 0 + Phase 1 Verification

Agent: go-diagnostics
Stage: Scan (Go)
Date: 2026-04-18

---

## 1. Summary of Changes

Phase 0 + Phase 1 added the full identity layer on top of the existing Organizations CRUD.
The implementation report (`02-implement-go.md`) states `go build ./... — SUCCESS`, `go vet ./... — SUCCESS`, `go test ./... — 76 passed, 0 failed`. These numbers reflect the state at implementation time and must be re-verified now, especially since Phase 2 and Phase 3 were subsequently landed in the same repository (commits `6f33a3d` and `8f5a708`).

---

## 2. Files Added or Modified (Phase 1 scope)

### Config
- `services/api-dashboard/internal/config/config.go` — replaced `JWTSecret string` with `BetterAuthJWKSURL string`; both `DATABASE_URL` and `BETTERAUTH_JWKS_URL` are now required at startup

### Middleware
- `services/api-dashboard/internal/middleware/auth.go` — HMAC keyfunc removed; replaced with `MicahParks/keyfunc/v3` JWKS RS256 keyfunc; added `ctxUserID`, `ctxOrgClaims` context keys; added `UserIDFromContext`, `OrgClaimsFromContext` helpers
- `services/api-dashboard/internal/middleware/testing.go` — new file: `InjectTestContext` helper for handler tests (bypasses JWKS)
- `services/api-dashboard/internal/middleware/auth_test.go` — rewritten: HMAC token helpers removed; covers header rejection, RequireRole, context helper zero/set values

### Models
- `services/api-dashboard/internal/model/user.go` — new: `User` struct, `OrgMembership` struct
- `services/api-dashboard/internal/model/org_invite.go` — new: `OrgInvite` struct

### Repositories
- `services/api-dashboard/internal/repository/user.go` — new: `UserRepository` interface + `userRepo` wrapping sqlc `GetUserByID`, `UpsertUser`
- `services/api-dashboard/internal/repository/org_invite.go` — new: `OrgInviteRepository` interface + `orgInviteRepo` wrapping sqlc `InsertOrgInvite`, `GetOrgInviteByToken`

### Services
- `services/api-dashboard/internal/service/user.go` — new: `UserService` with `GetMe` (DB lookup by userID) and `UpsertUser` (email validation + sqlc upsert); OTel spans; slog
- `services/api-dashboard/internal/service/org_invite.go` — new: `OrgInviteService.Create`; role validation (owner blocked); `crypto/rand` 32-byte base64url token; 7-day TTL

### Handlers
- `services/api-dashboard/internal/handler/user.go` — new: `UserHandler.GetMe`; reads userID from context, calls service, merges with `OrgClaimsFromContext`
- `services/api-dashboard/internal/handler/org_invite.go` — new: `OrgInviteHandler.Create`; parses UUID from chi path param `{id}`, decodes body, delegates to service
- `services/api-dashboard/internal/handler/organization_test.go` — rewritten: removed HMAC helpers; uses `middleware.InjectTestContext`

### Router
- `services/api-dashboard/internal/router/router.go` — prefix changed `/api/v1` → `/v1`; `User` and `OrgInvite` added to `Handlers` struct; `GET /v1/me` (ValidateJWT, no RequireRole); `POST /v1/orgs/{id}/invites` (RequireRole admin, owner)

### Main
- `services/api-dashboard/cmd/server/main.go` — wires `UserRepo`, `UserService`, `UserHandler`, `OrgInviteRepo`, `OrgInviteService`, `OrgInviteHandler`; `middleware.NewAuth(cfg.BetterAuthJWKSURL)` with `os.Exit(1)` on failure

### Tests (new)
- `services/api-dashboard/internal/service/user_test.go` — `TestUserService_GetMe` (3 cases: found, not found, db error); `TestUserService_UpsertUser` (3 cases: new user, existing user, empty email)
- `services/api-dashboard/internal/service/org_invite_test.go` — `TestOrgInviteService_Create` (6 cases: valid editor, valid admin, owner rejected, invalid role, empty email, db error)
- `services/api-dashboard/internal/handler/user_test.go` — `TestUserHandler_GetMe` (5 cases: with orgs, empty orgs, not found, db error, multiple orgs)
- `services/api-dashboard/internal/handler/org_invite_test.go` — `TestOrgInviteHandler_Create` (8 cases: editor/admin/viewer valid, invalid UUID, invalid JSON, empty email, owner role, db error)

### SQL Artifacts (Phase 1)
- Migrations added: `000002_create_users`, `000003_create_org_memberships`, `000004_create_org_invites` (all `.up.sql` and `.down.sql` present)
- sqlc query files: `packages/shared-domain/queries/users.sql`, `org_memberships.sql`, `org_invites.sql`
- sqlc generated code: `packages/shared-domain/db/users.sql.go`, `org_memberships.sql.go`, `org_invites.sql.go`

### Dependency Added
- `github.com/MicahParks/keyfunc/v3 v3.8.0` — JWKS keyfunc adapter for `golang-jwt/jwt/v5`

---

## 3. Affected Areas by Layer

| Layer | Affected | Notes |
|-------|----------|-------|
| Config | YES | `JWTSecret` gone; `BETTERAUTH_JWKS_URL` required |
| Middleware (JWT) | YES | Algorithm changed from HS256 → RS256/JWKS |
| Middleware (RBAC) | YES | `RequireRole` unchanged logic, but tests rewritten |
| Models | YES | 2 new model files |
| Repositories | YES | 2 new repos |
| Services | YES | 2 new services |
| Handlers | YES | 2 new handlers, org handler tests rewritten |
| Router | YES | Prefix change, 2 new route groups |
| Main (wiring) | YES | All new DI wired |
| Migrations | YES | 3 new migrations (000002–000004) |
| sqlc | YES | 3 new query files + generated code |

---

## 4. What Needs Testing

### Unit Tests (Go) — must pass `go test ./...`

| Package | Test function | Cases |
|---------|---------------|-------|
| `internal/middleware` | `TestAuth_ValidateJWT_HeaderChecks` | missing header, non-Bearer prefix |
| `internal/middleware` | `TestAuth_RequireRole` | exact match owner/admin/editor, viewer rejected, empty role, no allowed roles |
| `internal/middleware` | `TestContextHelpers_ZeroValues` | OrgID, Role, OrgIDs, UserID all return zero when unset |
| `internal/middleware` | `TestContextHelpers_SetValues` | all helpers return set values |
| `internal/service` | `TestUserService_GetMe` | found, not found (ErrNotFound), db error |
| `internal/service` | `TestUserService_UpsertUser` | new user, existing user, empty email (ErrInvalidInput) |
| `internal/service` | `TestOrgInviteService_Create` | valid editor, valid admin, owner rejected, invalid role, empty email, db error |
| `internal/handler` | `TestUserHandler_GetMe` | with orgs, empty orgs, not found → 404, db error → 500, multiple orgs |
| `internal/handler` | `TestOrgInviteHandler_Create` | editor/admin/viewer → 201, invalid UUID → 400, invalid JSON → 400, empty email → 400, owner → 400, db error → 500 |

### Build and Static Analysis
- `go build ./...` — must succeed across entire monorepo (including Phase 2+3 additions)
- `go vet ./...` — must report zero issues

### Key Behavioral Scenarios to Verify

1. **Route prefix**: `GET /v1/me` returns user profile; `GET /api/v1/me` returns 404
2. **JWT algorithm enforcement**: RS256 token accepted; HS256 token rejected with 401
3. **X-Org-ID membership check**: token with org `A` + `X-Org-ID: B` → 403
4. **X-Org-ID missing**: → 400 `MISSING_ORG_ID`
5. **GET /v1/me — user not found**: returns 404 `NOT_FOUND` (not 500)
6. **POST /v1/orgs/{id}/invites — role=owner**: returns 400 `INVALID_INPUT`
7. **POST /v1/orgs/{id}/invites — viewer caller**: blocked by RequireRole → 403 `FORBIDDEN`
8. **POST /v1/orgs/{id}/invites — non-UUID path**: returns 400 `INVALID_ID`

### Known Gap Noted in Implementation Report

The implementation report notes: "The current implementation only does `GetByID`. If the upsert-on-me flow is desired, the handler should call `UpsertUser` when `GetMe` returns `ErrNotFound`."

Current behavior: `GET /v1/me` for a new BetterAuth user who has never called this endpoint will return 404. Spec FR-7 says the user record should be upserted on first login. This is a functional gap — `UserService.GetMe` does NOT call `UpsertUser` as a fallback. The `UpsertUser` method exists and is tested but is not called from `GetMe` or the handler.

---

## 5. New UI Features Needing E2E Smoke Scenarios

All of the following are new in this phase (apps/dashboard was greenfield):

| Route | What to smoke test |
|-------|--------------------|
| `/login` | Form renders, valid credentials redirect to `/`, invalid credentials show toast |
| `/signup` | Form renders, new user creation redirects to `/onboarding` |
| `/onboarding` | 3-step wizard: org type selection → org name + slug → success screen; calls `POST /v1/organizations` |
| `/accept-invite/[token]` | Stub page renders with "coming soon" message and token display |
| `/` (dashboard) | Authenticated user sees shell layout (sidebar + topbar + org switcher) |
| Unauthenticated access to `/` | Redirects to `/login` |
| Org switcher | Renders orgs from session; selecting an org updates state |
| Logout | Clears session, redirects to `/login` |

---

## 6. Observations and Risk Areas

1. **JWKS hard dependency at startup**: `middleware.NewAuth(jwksURL)` calls `keyfunc.NewDefault` which attempts to fetch the JWKS endpoint immediately. If BetterAuth is not running during `go test` runs that construct a real `Auth` struct, tests will fail or hang. Handler and service tests use `InjectTestContext` / `newAuthForTest()` (nil JWKS) to avoid this — correct pattern used.

2. **`GET /v1/me` does not require `RequireRole`**: per spec (FR-4: "no RequireRole; any authenticated user") — confirmed correct in router. However, `ValidateJWT` still enforces `X-Org-ID` membership check. A user who has no orgs at all (immediately after signup, before onboarding) will receive 403 because `role == ""`. This is a potential UX issue for the post-signup flow, but is consistent with spec constraints.

3. **No test for JWKS live validation path**: `TestAuth_ValidateJWT_HeaderChecks` only covers pre-JWKS header failures. There is no integration test for the RS256 token parsing path (by design — requires running BetterAuth). The test-runner will not be able to exercise this path without a BETTERAUTH_JWKS_URL env var pointing to a real or mock JWKS.

4. **Route prefix change is breaking for any hardcoded `/api/v1` clients**: confirmed by AC-7 that old prefix must return 404. All existing test files (`organization_test.go`) must use `/v1` paths — this was addressed in the rewrite.

5. **Phase 2 and Phase 3 code also present**: the repository contains `publisher_apps`, `api_keys`, `publisher_rules`, `campaigns`, `creatives` (migrations 000005–000009, corresponding Go layers). The `go test ./...` run will cover all of these. Phase 1 scan is limited to Phase 1 files, but test-runner should run the full suite.

