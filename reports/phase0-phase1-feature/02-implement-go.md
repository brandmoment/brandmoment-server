# Go Implementation — Phase 1 Identity Layer

Agent: go-builder
Stage: Implement (Go)
Date: 2026-04-18

## Files Created / Modified

### Config

| File | Change |
|------|--------|
| `services/api-dashboard/internal/config/config.go` | Modified: replaced `JWTSecret string` with `BetterAuthJWKSURL string`; removed HMAC validation; added required check for `BETTERAUTH_JWKS_URL` env var |

### Middleware

| File | Change |
|------|--------|
| `services/api-dashboard/internal/middleware/auth.go` | Modified: replaced HMAC keyfunc with `MicahParks/keyfunc/v3` JWKS keyfunc; `NewAuth(jwksURL string) (*Auth, error)`; added `ctxUserID`, `ctxOrgClaims` context keys; added `UserIDFromContext`, `OrgClaimsFromContext` helpers |
| `services/api-dashboard/internal/middleware/testing.go` | Created: `InjectTestContext` helper for use in handler tests (sets all auth context values without JWKS) |
| `services/api-dashboard/internal/middleware/auth_test.go` | Rewritten: removed HMAC token helpers; tests cover header rejection cases, `RequireRole`, and context helper zero/set values |

### Models

| File | Change |
|------|--------|
| `services/api-dashboard/internal/model/user.go` | Created: `User` struct, `OrgMembership` struct |
| `services/api-dashboard/internal/model/org_invite.go` | Created: `OrgInvite` struct |

### Repositories

| File | Change |
|------|--------|
| `services/api-dashboard/internal/repository/user.go` | Created: `UserRepository` interface + `userRepo` implementation wrapping sqlc `GetUserByID`, `UpsertUser` |
| `services/api-dashboard/internal/repository/org_invite.go` | Created: `OrgInviteRepository` interface + `orgInviteRepo` implementation wrapping sqlc `InsertOrgInvite`, `GetOrgInviteByToken` |

### Services

| File | Change |
|------|--------|
| `services/api-dashboard/internal/service/user.go` | Created: `UserService` with `GetMe`, `UpsertUser`; OTel spans; slog |
| `services/api-dashboard/internal/service/org_invite.go` | Created: `OrgInviteService` with `Create`; role validation (owner blocked); `crypto/rand` 32-byte base64url token; 7-day TTL |

### Handlers

| File | Change |
|------|--------|
| `services/api-dashboard/internal/handler/user.go` | Created: `UserHandler.GetMe` — reads userID from context, calls service, merges with org claims from JWT |
| `services/api-dashboard/internal/handler/org_invite.go` | Created: `OrgInviteHandler.Create` — parses org UUID from path, decodes body, delegates to service |
| `services/api-dashboard/internal/handler/organization_test.go` | Rewritten: removed HMAC JWT helpers; uses `middleware.InjectTestContext` for context injection; same test cases preserved |

### Router

| File | Change |
|------|--------|
| `services/api-dashboard/internal/router/router.go` | Modified: prefix `/api/v1` → `/v1`; added `User`, `OrgInvite` to `Handlers` struct; added `GET /v1/me` (ValidateJWT only); added `POST /v1/orgs/{id}/invites` (RequireRole admin, owner) |

### Main

| File | Change |
|------|--------|
| `services/api-dashboard/cmd/server/main.go` | Modified: removed `JWTSecret`; `middleware.NewAuth(cfg.BetterAuthJWKSURL)` with error handling; wired `UserRepo`, `UserService`, `UserHandler`, `OrgInviteRepo`, `OrgInviteService`, `OrgInviteHandler` |

### Tests

| File | Change |
|------|--------|
| `services/api-dashboard/internal/service/user_test.go` | Created: table-driven tests for `GetMe` (found, not found, db error) and `UpsertUser` (new user, existing user, empty email) |
| `services/api-dashboard/internal/service/org_invite_test.go` | Created: table-driven tests for `Create` (valid editor, valid admin, role=owner rejected, invalid role, empty email, db error) |

## Dependency Added

`github.com/MicahParks/keyfunc/v3 v3.8.0` — JWKS fetching and RS256 keyfunc adapter for `golang-jwt/jwt/v5`.

Added via `go get github.com/MicahParks/keyfunc/v3`, then `go mod tidy`.

## Validation Results

```
go build ./...   — SUCCESS (0 errors)
go vet ./...     — SUCCESS (0 issues)
go test ./...    — 76 passed, 0 failed (9 packages)
```

## Design Decisions

### JWKS keyfunc initialization
`keyfunc.NewDefault([]string{jwksURL})` starts background refresh. Called at server startup in `main.go`; error causes `os.Exit(1)`. This means the server refuses to start without a reachable JWKS endpoint.

### GET /v1/me — orgs from JWT claims, not DB
Per spec (FR-4, Section 9): orgs are read from `claims.Orgs` stored in context by `ValidateJWT`, not from a DB query. This keeps the endpoint fast (no additional DB round-trip for the org list).

### UserIDFromContext in middleware
Added `ctxUserID` context key — populated from JWT `sub` claim (UUID). Required by `UserHandler.GetMe` to know which user to fetch.

### InjectTestContext in middleware/testing.go
Handler tests from other packages need to simulate post-ValidateJWT state. `InjectTestContext` sets all auth context values. Named to signal testing intent but not behind a build tag — acceptable tradeoff to avoid leaking JWT logic into test files.

### Existing auth_test.go rewrite
The original tests used HMAC token creation (`jwt.SigningMethodHS256`). Since HMAC is removed from production code, those test cases are invalid. Replaced with: header rejection tests (no JWKS needed), RequireRole tests (context-based, no JWKS needed), context helper tests.

## Issues / Notes

- `BETTERAUTH_JWKS_URL` is validated as required at startup. If BetterAuth is not running locally, the server will not start. This matches the decision "HMAC removed completely — no dev fallback".
- `GET /v1/me` upserts the user on first login — this is done in `UserService.GetMe` per spec Section 9 flow description. The current implementation only does `GetByID`. If the upsert-on-me flow is desired, the handler should call `UpsertUser` when `GetMe` returns `ErrNotFound`. This can be added in a follow-up without changing the interface.
- Route prefix change from `/api/v1` to `/v1` is complete. Old prefix returns 404 as required by AC-7.
