---
description: Multi-tenancy rules — org_id filtering, JWT claims, RBAC middleware for Go services
globs: "**/*.go"
---

# Multi-Tenancy Rules

## org_id Filtering

Sub-resources (campaigns, publisher_apps, api_keys, targeting_rules) → ALWAYS `WHERE org_id = $1 AND id = $2`.
Top-level (organizations, users) → NO org_id column. Access via JWT membership / `org_memberships` join.

Rule: does the table have `org_id` FK? Yes → filter. No → access controlled otherwise.

## Auth Middleware

**ValidateJWT**: extract Bearer token → validate via `golang-jwt/jwt/v5` against JWKS → parse `orgs[]` → extract `X-Org-ID` header → verify membership → store org_id + role in context.

Context helpers: `OrgIDFromContext(ctx)`, `RoleFromContext(ctx)`.

**RequireRole**: middleware checking role from context against allowed roles list. Uses shared `respondError` helper.

## Security Rules

1. Every sub-resource query MUST include org_id — never return another org's data
2. Never trust client-sent org_id in body — use org_id from JWT/context
3. Never skip RBAC on mutations (POST/PUT/DELETE)
4. Admin org can access cross-org data with explicit `orgType == "admin"` check
