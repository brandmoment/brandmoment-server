# Go Tests — Phase 1 Identity Layer

Agent: go-test-writer
Stage: Test
Date: 2026-04-18

## Summary

Added handler-level tests for the two new Phase 1 handlers. All existing tests continue to pass.

| Metric | Before | After |
|--------|--------|-------|
| Total tests passing | 76 | 92 |
| New tests added | — | 16 |
| Packages covered | 9 | 9 (handler package expanded) |

## Coverage Review

### Existing tests — verified working

| File | Status | Notes |
|------|--------|-------|
| `service/user_test.go` | PASS | 3 GetMe cases + 3 UpsertUser cases; covers found/not-found/db-error and empty-email validation |
| `service/org_invite_test.go` | PASS | 6 cases: valid editor, valid admin, owner-blocked, invalid-role, empty-email, db-error |
| `middleware/auth_test.go` | PASS | Rewritten by go-builder: header rejection, RequireRole, context helpers zero/set values. No JWKS needed — header checks fire before token parsing |
| `handler/organization_test.go` | PASS | Create/GetByID/List + handleServiceError; uses `middleware.InjectTestContext`; adapted to `/v1` prefix |

### New tests created

#### `handler/user_test.go` — `TestUserHandler_GetMe`

5 table-driven cases covering `UserHandler.GetMe`:

| Case | Verifies |
|------|---------|
| user found, org claims present | 200 + correct email/name + orgs slice length = 1 |
| user found, no org IDs in context | 200 + orgs slice length = 0 |
| user not found | 404 NOT_FOUND |
| service db error (DeadlineExceeded) | 500 INTERNAL_ERROR |
| multiple org claims | 200 + orgs slice length = 3 |

Pattern: mock `repository.UserRepository` → `service.NewUserService` → `handler.NewUserHandler`. Context injected via `middleware.InjectTestContext` (sets userID, orgID, role, orgClaims). No chi URL param needed for this handler.

#### `handler/org_invite_test.go` — `TestOrgInviteHandler_Create`

9 table-driven cases covering `OrgInviteHandler.Create`:

| Case | Verifies |
|------|---------|
| valid editor role | 201 + non-empty token + correct email/role |
| valid admin role | 201 |
| valid viewer role | 201 |
| invalid org UUID in path | 400 INVALID_ID |
| invalid JSON body | 400 INVALID_BODY |
| empty email | 400 INVALID_INPUT (service validation) |
| owner role | 400 INVALID_INPUT (owner blocked at service) |
| invalid role string | 400 INVALID_INPUT |
| db error | 500 INTERNAL_ERROR |

Pattern: mock `repository.OrgInviteRepository` → `service.NewOrgInviteService` → `handler.NewOrgInviteHandler`. Chi URL param `"id"` injected via existing `withChiID` helper from `organization_test.go` (same package, accessible without redefinition).

## Design Notes

- Handler tests use concrete service constructors with mock repos (not mock services). This tests the full handler→service chain without a database, which is the correct approach given services are concrete structs not interfaces in this codebase.
- `middleware.InjectTestContext` is the sole mechanism for setting JWT-derived context values in handler tests; no JWKS server needed.
- `withChiID` helper is reused from `organization_test.go` — same `handler` package, no duplication.
- The "empty org IDs" case for GetMe passes `[]uuid.UUID{}` to `InjectTestContext`. `InjectTestContext` iterates orgIDs to build OrgClaims — empty input produces empty claims slice, verified by asserting `len(orgs) == 0` in response.
- All test files follow the project convention: table-driven, `t.Run`, no testify/gomock.

## Validation

```
go build ./...   — SUCCESS (0 errors)
go vet ./...     — SUCCESS (0 issues)
go test ./...    — 92 passed, 0 failed (9 packages)
```
