# Research Report: JWT Authorization Flow — Request to DB

Date: 2026-04-18
Status: Complete

---

## Summary

The BrandMoment platform uses a JWT-based multi-tenant authorization system for the `/api/v1/organizations` endpoints. The flow chains HTTP middleware → handler → service → repository → sqlc-generated DB queries. Two distinct access patterns are implemented:

1. **Top-level resources (organizations)** — access controlled via JWT membership, membership check at handler layer
2. **Sub-resources (campaigns, api_keys, etc.)** — access controlled via `org_id` in SQL WHERE clause

The implementation has **4 CRITICAL security gaps** and **6 HIGH severity findings**, primarily around JWT validation (symmetric HMAC instead of asymmetric JWKS, missing algorithm check, missing issuer/audience validation) and broken access control (viewer role can create organizations).

---

## 1. Complete JWT Auth Flow: Request to Database

### 1.1 HTTP Request Entry

```
Client HTTP Request:
  Authorization: Bearer <jwt_token>
  X-Org-ID: <uuid>
  POST/GET /api/v1/organizations[/{id}]
```

### 1.2 Middleware Chain (router.go:20-37)

Chain order in `services/api-dashboard/internal/router/router.go`:

```
otelchi.Middleware("api-dashboard")          — OTel root span per request
  ↓
chimiddleware.RequestID                       — X-Request-ID injection
  ↓
chimiddleware.RealIP                          — RemoteAddr from X-Real-IP
  ↓
chimiddleware.Recoverer                       — panic recovery
  ↓
auth.ValidateJWT [middleware/auth.go:40-95]   — JWT extraction, parsing, membership check
  ↓
auth.RequireRole(...) [middleware/auth.go:97-110]  — role-based gate
  ↓
Handler (HTTP layer)
```

### 1.3 ValidateJWT Middleware Execution

**File:** `services/api-dashboard/internal/middleware/auth.go:40-95`

**Step 1: Token Extraction (lines 42-48)**
```go
authHeader := r.Header.Get("Authorization")
if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
    httputil.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", ...)
    return
}
tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
```

**Step 2: JWT Parsing (lines 50-57)**
```go
claims := &Claims{}
token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
    return a.jwtSecret, nil  // Returns symmetric HMAC secret — NO algorithm check
})
if err != nil || !token.Valid {
    httputil.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
    return
}
```

**Claims Structure:**
```go
type Claims struct {
    jwt.RegisteredClaims
    Orgs []OrgClaim `json:"orgs"`
}
type OrgClaim struct {
    OrgID string `json:"org_id"`
    Role  string `json:"role"`
}
```

**Step 3: Parse X-Org-ID Header (lines 59-69)**
```go
xOrgID := r.Header.Get("X-Org-ID")
if xOrgID == "" {
    httputil.RespondError(w, http.StatusBadRequest, "MISSING_ORG_ID", ...)
    return
}
orgID, err := uuid.Parse(xOrgID)
if err != nil {
    httputil.RespondError(w, http.StatusBadRequest, "INVALID_ORG_ID", ...)
    return
}
```

**Step 4: Validate Membership (lines 71-87)**
```go
var role string
var orgIDs []uuid.UUID
for _, oc := range claims.Orgs {
    parsed, err := uuid.Parse(oc.OrgID)
    if err != nil {
        continue  // Silently skip unparseable org IDs — no logging
    }
    orgIDs = append(orgIDs, parsed)
    if parsed == orgID {
        role = oc.Role  // Found active org's role
    }
}
if role == "" {
    httputil.RespondError(w, http.StatusForbidden, "FORBIDDEN", 
        "you are not a member of this organization")
    return
}
```

Builds two pieces of data:
- `orgIDs []uuid.UUID` — all organizations from JWT claims
- `role string` — the user's role in the requested org (X-Org-ID)

**Step 5: Store in Context (lines 89-93)**
```go
ctx = context.WithValue(ctx, ctxOrgID, orgID)    // "org_id" key
ctx = context.WithValue(ctx, ctxRole, role)      // "role" key
ctx = context.WithValue(ctx, ctxOrgIDs, orgIDs)  // "org_ids" key
next.ServeHTTP(w, r.WithContext(ctx))
```

| Context Key | Type | Value |
|---|---|---|
| `org_id` | `uuid.UUID` | Active org from X-Org-ID |
| `role` | `string` | User's role in active org |
| `org_ids` | `[]uuid.UUID` | All orgs user belongs to |

### 1.4 RequireRole Middleware

**File:** `services/api-dashboard/internal/middleware/auth.go:97-110`

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
            httputil.RespondError(w, http.StatusForbidden, "FORBIDDEN", 
                "insufficient permissions")
        })
    }
}
```

Linear scan of allowed roles. No special handling for cross-org admin access.

### 1.5 Handler Layer

**File:** `services/api-dashboard/internal/handler/organization.go`

#### GetByID (lines 42-63)
```go
func (h *OrganizationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := uuid.Parse(idStr)
    if err != nil {
        httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", ...)
        return
    }

    orgIDs := middleware.OrgIDsFromContext(r.Context())
    if !slices.Contains(orgIDs, id) {
        httputil.RespondError(w, http.StatusForbidden, "FORBIDDEN", ...)
        return
    }

    org, err := h.service.GetByID(r.Context(), id)
    // ...
}
```

Explicit membership check: requested `{id}` must exist in user's `orgIDs` slice before DB call.

#### List (lines 65-75)
```go
orgIDs := middleware.OrgIDsFromContext(r.Context())
orgs, err := h.service.ListByIDs(r.Context(), orgIDs)
```

Passes full `orgIDs` slice to service.

#### Create (lines 26-40)
```go
var req service.CreateOrganizationRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil { ... }
org, err := h.service.Create(r.Context(), req)
```

Does NOT pass `org_id` from context. New organization gets `uuid.New()` in service layer.

### 1.6 Service Layer

**File:** `services/api-dashboard/internal/service/organization.go`

```go
func (s *OrganizationService) GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
    ctx, span := s.tracer.Start(ctx, "OrganizationService.GetByID")
    defer span.End()

    slog.InfoContext(ctx, "getting organization", slog.String("id", id.String()))

    org, err := s.repo.GetByID(ctx, id)
    if err != nil {
        span.RecordError(err)
        return nil, err  // Propagates sentinel errors unwrapped
    }
    return org, nil
}
```

- OTel span started, context passed to repo
- `slog.InfoContext` for structured logging with trace correlation
- Errors returned unwrapped (preserves sentinel error type for handler's `errors.Is` checks)

### 1.7 Repository Layer

**File:** `services/api-dashboard/internal/repository/organization.go:46-55`

```go
func (r *organizationRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
    row, err := r.q.GetOrganizationByID(ctx, uuidToPgtype(id))
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, model.ErrNotFound  // Sentinel error
        }
        return nil, fmt.Errorf("get organization: %w", err)
    }
    return toOrganization(row), nil
}
```

- Wraps sqlc-generated `r.q.GetOrganizationByID(ctx, ...)`
- Maps `pgx.ErrNoRows` → `model.ErrNotFound` (sentinel)
- Type conversion: `pgtype.UUID` ↔ `uuid.UUID`

### 1.8 sqlc Query Layer

**File:** `packages/shared-domain/queries/organizations.sql`

```sql
-- name: GetOrganizationByID :one
SELECT id, type, name, slug, created_at, updated_at
FROM organizations
WHERE id = @id;
```

No `org_id` filter — correct, because `organizations` is a top-level resource. Access control is at handler layer via membership check.

**Generated Go** (`packages/shared-domain/db/organizations.sql.go`):
```go
func (q *Queries) GetOrganizationByID(ctx context.Context, id pgtype.UUID) (Organization, error) {
    row := q.db.QueryRow(ctx, getOrganizationByID, id)
    var i Organization
    err := row.Scan(&i.ID, &i.Type, &i.Name, &i.Slug, &i.CreatedAt, &i.UpdatedAt)
    return i, err
}
```

OTel context with active span passed through, enabling DB-level tracing.

### 1.9 Complete Call Chain: GET /api/v1/organizations/{id}

```
HTTP Request (Authorization: Bearer <jwt>, X-Org-ID: <uuid>)
  ↓
otelchi + RequestID + RealIP + Recoverer
  ↓
ValidateJWT:
  • jwt.ParseWithClaims(token, claims, keyFunc) — validates HMAC signature
  • Extracts claims.Orgs []OrgClaim
  • Builds orgIDs[] from all claims.Orgs
  • Reads X-Org-ID, finds role in claims.Orgs
  • context.WithValue(org_id, role, org_ids)
  ↓
RequireRole("viewer","editor","admin","owner"):
  • RoleFromContext(ctx) vs allowed list
  • Returns 403 if role not in list
  ↓
GetByID Handler:
  • Extracts {id} from URL parameter
  • OrgIDsFromContext(ctx) → []uuid.UUID
  • slices.Contains(orgIDs, id) → 403 if not found
  • h.service.GetByID(ctx, id)
  ↓
Service:
  • s.tracer.Start(ctx, "OrganizationService.GetByID")
  • slog.InfoContext(ctx, ...)
  • s.repo.GetByID(ctx, id)
  ↓
Repository:
  • uuidToPgtype(id) → pgtype.UUID
  • r.q.GetOrganizationByID(ctx, ...)
  ↓
sqlc-generated Query:
  • q.db.QueryRow(ctx, "SELECT ... FROM organizations WHERE id = $1", id)
  • row.Scan(&org)
  • if pgx.ErrNoRows → model.ErrNotFound
  ↓
Handler Response:
  • httputil.RespondJSON(w, http.StatusOK, org)
  • {"data": {"id": "...", "type": "...", ...}}
```

---

## 2. Two Access Patterns: Top-Level vs Sub-Resources

### 2.1 Pattern A: Top-Level Resources (organizations)

**Characteristics:**
- No `org_id` column in table
- Resource IS an organization
- Access via JWT membership array

**Flow:**
```
Handler receives X-Org-ID header (requested org)
  ↓
ValidateJWT extracts claims.Orgs[] (all user's orgs) + role for requested org
  ↓
Handler gets orgIDs[] from context (all user's orgs)
  ↓
Handler explicitly checks: slices.Contains(orgIDs, id)
  ↓
SQL query: WHERE id = $1 (no org_id filter)
```

**Example: GetByID**
```go
// handler/organization.go:50-54
orgIDs := middleware.OrgIDsFromContext(r.Context())
if !slices.Contains(orgIDs, id) {
    httputil.RespondError(w, http.StatusForbidden, "FORBIDDEN", ...)
    return
}
org, err := h.service.GetByID(r.Context(), id)
```

Membership verification at handler layer, before service/repo call.

**Example: List**
```go
// handler/organization.go:70
orgIDs := middleware.OrgIDsFromContext(r.Context())
orgs, err := h.service.ListByIDs(r.Context(), orgIDs)
```

SQL: `WHERE id = ANY($1::uuid[])`

### 2.2 Pattern B: Sub-Resources (when implemented)

**Characteristics:**
- `org_id` column in table
- Resource belongs to an organization
- Access via active org from context + RBAC
- Filtering at SQL layer

**Expected Flow:**
```
Handler receives {id} in URL or body
  ↓
Handler gets orgID (active org) from context: middleware.OrgIDFromContext(ctx)
  ↓
Handler does NOT verify membership (middleware did that already)
  ↓
Handler passes orgID to service: h.service.GetByID(ctx, orgID, resourceID)
  ↓
Service passes to repo: s.repo.GetByID(ctx, orgID, resourceID)
  ↓
SQL query: WHERE org_id = $1 AND id = $2
```

**Why the difference?**
- organizations → no org_id column → membership check at handler
- campaigns, api_keys, etc. → org_id column → filter at SQL layer

Both patterns result in the same security outcome (user cannot access another org's data), but approach it differently.

---

## 3. Security Findings Summary

### 3.1 CRITICAL Severity (Fix Immediately)

| ID | Issue | File | Impact |
|---|---|---|---|
| **01** | JWT validated with symmetric HMAC secret instead of JWKS | middleware/auth.go:51-56 | BetterAuth tokens NOT validated. Server accepts any HMAC-signed token. |
| **02** | No signing algorithm check in key function | middleware/auth.go:51-56 | Algorithm confusion attack possible (`alg:none` bypass depending on library version). |
| **03** | Viewer role can POST (create organizations) | router/router.go:31-36 | Broken access control. Viewer role should be read-only. |
| **04** | No issuer/audience validation on JWT | middleware/auth.go:51-56 | Cross-service token replay. Any token signed with same secret accepted. |

### 3.2 HIGH Severity (Fix Soon)

| ID | Issue | File | Impact |
|---|---|---|---|
| **05** | JWT expiration not explicitly required | middleware/auth.go:51-56 | Tokens without `exp` claim never expire. |
| **06** | Unbounded org_id array in ListByIDs query | repository/organization.go | DoS: token with thousands of org_ids causes large SQL query. |
| **07** | Service error messages leaked to HTTP response | handler/organization.go:77-88 | Information disclosure: internal operation names exposed. |

### 3.3 MEDIUM Severity

- No rate limiting on any endpoint (brute-force vulnerability)
- No request body size limit (memory exhaustion)
- No CORS configuration
- OTel exporter uses plaintext gRPC (trace data unencrypted)
- Hardcoded MinIO credentials in seed tool

### 3.4 LOW Severity

- Silent skip of unparseable JWT org_id claims (no logging)
- Context accessor functions return zero value silently (no panic guard)
- No Content-Type validation on incoming requests

**Severity Distribution:**
- 4 CRITICAL
- 3 HIGH
- 5 MEDIUM
- 3 LOW

---

## 4. What Is Implemented Correctly

### 4.1 Architectural Correctness

1. **X-Org-ID validation against JWT** — header value verified to exist in claims.Orgs array before storing in context. Prevents impersonation.

2. **org_id never from request body** — always sourced from JWT context (ValidateJWT stores it). Enforced at handler layer.

3. **Top-level vs sub-resource distinction** — organizations correctly has no org_id column; pattern documented for future sub-resources.

4. **Handler membership check for top-level** — explicit `slices.Contains(orgIDs, id)` check before service call for organizations.

5. **sqlc for all DB queries** — no raw SQL strings in application code. Eliminates SQL injection vectors.

6. **Sentinel error pattern** — `model.ErrNotFound`, `ErrInvalidInput`, `ErrUnauthorized` returned unwrapped to preserve type information for handler's `errors.Is` checks.

7. **Single source of truth for HTTP responses** — `httputil.RespondJSON` and `httputil.RespondError` functions used by all handlers and middleware. No duplicate error response logic.

### 4.2 Observability

1. **OTel span propagation** — context carries active span from middleware through handler → service → repository → DB query.

2. **Structured logging with slog** — `slog.InfoContext(ctx, ...)` with typed attributes. OTel trace ID correlation automatic.

3. **Span error recording** — `span.RecordError(err)` at service layer.

### 4.3 Error Handling

1. **pgx.ErrNoRows → sentinel mapping** — repository correctly maps no-row result to `ErrNotFound` (not leaking pgx-specific error).

2. **Error wrapping with context** — repository wraps other DB errors with `fmt.Errorf("operation: %w", err)` for diagnostics.

3. **Service error propagation** — errors returned unwrapped when they are sentinels, wrapped when adding context.

---

## 5. Key Files Reference

| File | Purpose | Key Functions |
|---|---|---|
| `services/api-dashboard/internal/middleware/auth.go` | JWT validation, context setup | `ValidateJWT`, `RequireRole`, context accessors |
| `services/api-dashboard/internal/router/router.go` | HTTP routing, middleware chain | `NewRouter` |
| `services/api-dashboard/internal/handler/organization.go` | HTTP request/response | `GetByID`, `List`, `Create` |
| `services/api-dashboard/internal/service/organization.go` | Business logic, OTel, slog | `GetByID`, `ListByIDs`, `Create` |
| `services/api-dashboard/internal/repository/organization.go` | DB access wrapper | `GetByID`, `ListByIDs` |
| `services/api-dashboard/internal/httputil/response.go` | HTTP response envelope | `RespondJSON`, `RespondError` |
| `services/api-dashboard/cmd/server/main.go` | Server startup, DI | Auth initialization, server setup |
| `services/api-dashboard/internal/config/config.go` | Environment config | `Config` struct, JWT_SECRET loading |
| `packages/shared-domain/queries/organizations.sql` | sqlc query definitions | GetOrganizationByID, ListOrganizationsByIDs |
| `packages/shared-domain/db/organizations.sql.go` | sqlc-generated Go code | Queries struct, scan methods |

---

## 6. Recommendations (Priority Order)

1. **Replace HMAC with JWKS** (CRITICAL) — Fetch BetterAuth JWKS at startup, validate against asymmetric public key, add algorithm check.

2. **Add issuer/audience validation** (CRITICAL) — Use `jwt.WithIssuer()` and `jwt.WithAudience()` parser options.

3. **Fix viewer POST access** (CRITICAL) — Apply `RequireRole` per-route, not per-group. `POST /organizations` should require `owner`.

4. **Add algorithm whitelist check** (CRITICAL) — Guard in key function prevents algorithm confusion attacks.

5. **Add JWT expiration requirement** (HIGH) — Pass `jwt.WithExpirationRequired()` to parser.

6. **Bound org_ids array** (HIGH) — Cap at 100 items; return 400 if JWT has more.

7. **Mask error messages** (HIGH) — Use static error strings in responses; log full errors server-side only.

8. **Add rate limiting** (MEDIUM) — Use `github.com/go-chi/httprate` on `/api/v1` group.

9. **Add request body size limit** (MEDIUM) — Wrap with `http.MaxBytesReader` or use chi middleware.

10. **Add CORS middleware** (MEDIUM) — Use `github.com/go-chi/cors` with explicit allowed origins.
