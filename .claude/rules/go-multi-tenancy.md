---
description: Multi-tenancy rules — org_id filtering, JWT claims, RBAC middleware for Go services
globs: "**/*.go"
---

# Multi-Tenancy Rules

## Org Types

3 organization types: `admin`, `publisher`, `brand`. Stored as TEXT with CHECK constraint in DB.

## JWT Claims Structure (BetterAuth)

```json
{
  "sub": "user_uuid",
  "orgs": [
    {"org_id": "550e8400-...", "role": "owner"},
    {"org_id": "6ba7b810-...", "role": "viewer"}
  ]
}
```

A user can belong to multiple orgs with different roles. The active org is determined by the `X-Org-ID` request header (validated against JWT memberships).

## Roles

`owner > admin > editor > viewer`

- **owner**: full access + delete org + manage billing
- **admin**: full access + invite/remove members
- **editor**: create/update resources (campaigns, apps, rules)
- **viewer**: read-only access

## When org_id Filtering Applies

### Sub-resources — ALWAYS filter by org_id

Tables that belong to an organization:
- `campaigns` → `WHERE org_id = $1 AND id = $2`
- `publisher_apps` → `WHERE org_id = $1 AND id = $2`
- `api_keys` → `WHERE org_id = $1 AND id = $2`
- `targeting_rules` → `WHERE org_id = $1 AND id = $2`

### Top-level resources — NO org_id column

Tables that ARE organizations or independent:
- `organizations` — no org_id column. Access via JWT membership: user sees only orgs from their `orgs[]` claim
- `users` — accessed through `org_memberships` table join

### How to decide

Ask: "Does this table have an `org_id` foreign key?"
- YES → filter by org_id in every query
- NO → access controlled by JWT membership or other means

Do NOT blindly add `WHERE org_id = $1` to every query.

## Auth Middleware Implementation

### ValidateJWT

1. Extract `Authorization: Bearer <token>` header
2. Validate token using `golang-jwt/jwt/v5` against BetterAuth JWKS
3. Parse `orgs` array from claims
4. Extract `X-Org-ID` header — the org the user wants to act on behalf of
5. Verify `X-Org-ID` exists in user's `orgs` array
6. Store org_id AND role in request context

```go
func OrgIDFromContext(ctx context.Context) uuid.UUID { ... }
func RoleFromContext(ctx context.Context) string { ... }
```

### RequireRole

```go
func (a *Auth) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromContext(r.Context())
			for _, allowed := range roles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			respondError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		})
	}
}
```

## Security Rules

1. NEVER return data from another org — every sub-resource query MUST include org_id
2. NEVER trust client-sent org_id in request body — always use org_id from JWT/context
3. NEVER skip RBAC on mutation endpoints (POST/PUT/DELETE)
4. Admin org can access cross-org data — add explicit check `if orgType == "admin"`