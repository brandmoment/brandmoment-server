# JWT Authorization Flow — Go Backend Trace

Agent: go-diagnostics
Date: 2026-04-18

---

## 1. Startup: Auth Initialization

**File:** `services/api-dashboard/cmd/server/main.go:59`

```go
auth := middleware.NewAuth(cfg.JWTSecret)
```

`cfg.JWTSecret` is loaded from the `JWT_SECRET` environment variable
(`services/api-dashboard/internal/config/config.go:17`). The secret is passed as a raw
`[]byte` to `Auth`. This means the service uses **HMAC symmetric validation** — NOT a
JWKS endpoint. Despite the architecture rules stating "validate against BetterAuth JWKS",
the actual implementation uses a shared secret stored in config.

---

## 2. Router: Middleware Chain Registration

**File:** `services/api-dashboard/internal/router/router.go`

Full middleware chain order for `/api/v1/organizations`:

```
otelchi.Middleware("api-dashboard")   // OTel tracing span per request
chimiddleware.RequestID               // injects X-Request-ID
chimiddleware.RealIP                  // sets RemoteAddr from X-Real-IP / X-Forwarded-For
chimiddleware.Recoverer               // panic recovery
  └── auth.ValidateJWT                // JWT extraction + org membership check (line 29)
        └── auth.RequireRole(...)     // role-based gate (line 32)
              └── handler             // actual handler function
```

Route definitions (router.go:31-36):
- `POST /api/v1/organizations` → `OrganizationHandler.Create`
- `GET  /api/v1/organizations` → `OrganizationHandler.List`
- `GET  /api/v1/organizations/{id}` → `OrganizationHandler.GetByID`

All three are behind `RequireRole("viewer", "editor", "admin", "owner")` — i.e. any
authenticated org member can call all three, including mutations.

---

## 3. ValidateJWT Middleware

**File:** `services/api-dashboard/internal/middleware/auth.go:40-95`

Step-by-step execution:

**Step 1 — Extract token (lines 42-48):**
```go
authHeader := r.Header.Get("Authorization")
if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
    httputil.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", ...)
    return
}
tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
```

**Step 2 — Parse and validate JWT (lines 50-57):**
```go
claims := &Claims{}
token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
    return a.jwtSecret, nil
})
if err != nil || !token.Valid {
    httputil.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
    return
}
```

The key function returns `a.jwtSecret` unconditionally — there is NO signing method check
(no `token.Method.Alg()` guard). This means the middleware would accept tokens signed with
any algorithm, including `alg: "none"`. This is a security gap relative to best practice
(see note in Section 9).

`Claims` struct (lines 22-30):
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

**Step 3 — Extract X-Org-ID header and parse (lines 59-69):**
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

**Step 4 — Validate membership and extract role (lines 71-87):**
```go
var role string
var orgIDs []uuid.UUID
for _, oc := range claims.Orgs {
    parsed, err := uuid.Parse(oc.OrgID)
    if err != nil {
        continue
    }
    orgIDs = append(orgIDs, parsed)
    if parsed == orgID {
        role = oc.Role
    }
}
if role == "" {
    httputil.RespondError(w, http.StatusForbidden, "FORBIDDEN", "you are not a member of this organization")
    return
}
```

This loop does two things simultaneously:
- Builds `orgIDs` — all org UUIDs from the JWT (used later for list/get operations)
- Checks if `X-Org-ID` exists in the JWT `orgs` array and captures the role

**Step 5 — Store in request context (lines 89-93):**
```go
ctx = context.WithValue(ctx, ctxOrgID, orgID)    // "org_id" key
ctx = context.WithValue(ctx, ctxRole, role)       // "role" key
ctx = context.WithValue(ctx, ctxOrgIDs, orgIDs)   // "org_ids" key
next.ServeHTTP(w, r.WithContext(ctx))
```

Three values stored:
| Context Key | Type           | Value                                   |
|-------------|----------------|-----------------------------------------|
| `org_id`    | `uuid.UUID`    | The active org from X-Org-ID            |
| `role`      | `string`       | The user's role in that org             |
| `org_ids`   | `[]uuid.UUID`  | All orgs the user belongs to (from JWT) |

---

## 4. RequireRole Middleware

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
            httputil.RespondError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
        })
    }
}
```

Reads `role` from context (set by `ValidateJWT`). Iterates the allowed roles slice
linearly. If no match → 403 FORBIDDEN. No special handling for admin cross-org access.

Context accessors (lines 112-125):
```go
func OrgIDFromContext(ctx context.Context) uuid.UUID {
    id, _ := ctx.Value(ctxOrgID).(uuid.UUID)
    return id
}
func RoleFromContext(ctx context.Context) string {
    role, _ := ctx.Value(ctxRole).(string)
    return role
}
func OrgIDsFromContext(ctx context.Context) []uuid.UUID {
    ids, _ := ctx.Value(ctxOrgIDs).([]uuid.UUID)
    return ids
}
```

All three use type assertion with blank discard — returns zero value if key missing or
wrong type (silent failure rather than panic).

---

## 5. Handler Layer

**File:** `services/api-dashboard/internal/handler/organization.go`

### GetByID (lines 42-63)

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
    ...
}
```

The handler performs an explicit membership check: the requested `{id}` must be in the
user's `orgIDs` slice from context. If not → 403 before any DB call. This is the correct
pattern for `organizations` since the table has no `org_id` column.

`org_id` from context (`OrgIDFromContext`) is NOT used in `GetByID` — membership
verification is done via `OrgIDsFromContext` instead. The active `org_id` (from X-Org-ID)
is not passed into the service for this endpoint.

### List (lines 65-75)

```go
orgIDs := middleware.OrgIDsFromContext(r.Context())
orgs, err := h.service.ListByIDs(r.Context(), orgIDs)
```

Passes the full `orgIDs` slice (all orgs from JWT) to the service. No further filtering.

### Create (lines 26-40)

```go
var req service.CreateOrganizationRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil { ... }
org, err := h.service.Create(r.Context(), req)
```

Does NOT pass `org_id` from context. The new organization gets a fresh `uuid.New()` in the
service layer. No org membership is created after insert (no `org_memberships` row).

### handleServiceError (lines 77-88)

```go
func handleServiceError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, model.ErrNotFound):
        httputil.RespondError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
    case errors.Is(err, model.ErrInvalidInput):
        httputil.RespondError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
    case errors.Is(err, model.ErrUnauthorized):
        httputil.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
    default:
        httputil.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
    }
}
```

Uses `errors.Is` (correct — works with wrapped errors via `fmt.Errorf("%w", ...)`).

---

## 6. Service Layer

**File:** `services/api-dashboard/internal/service/organization.go`

### GetByID (lines 70-83)

```go
func (s *OrganizationService) GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
    ctx, span := s.tracer.Start(ctx, "OrganizationService.GetByID")
    defer span.End()

    slog.InfoContext(ctx, "getting organization", slog.String("id", id.String()))

    org, err := s.repo.GetByID(ctx, id)
    if err != nil {
        span.RecordError(err)
        return nil, err      // propagates ErrNotFound as-is (no wrapping)
    }
    return org, nil
}
```

Note: `GetByID` returns `err` unwrapped. This is intentional — `ErrNotFound` is a sentinel
and must survive `errors.Is` checks in the handler. The service does not add context to the
error here (compare with `ListByIDs` which uses `fmt.Errorf("list organizations: %w", err)`).

### OTel tracing pattern used throughout:
```go
ctx, span := s.tracer.Start(ctx, "OrganizationService.<Method>")
defer span.End()
// ...
span.RecordError(err)
```

The `ctx` with the active span is passed down to `s.repo.*`, ensuring the DB calls appear
as child spans in Jaeger.

---

## 7. Repository Layer

**File:** `services/api-dashboard/internal/repository/organization.go`

### GetByID (lines 46-55)

```go
func (r *organizationRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
    row, err := r.q.GetOrganizationByID(ctx, uuidToPgtype(id))
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, model.ErrNotFound
        }
        return nil, fmt.Errorf("get organization: %w", err)
    }
    return toOrganization(row), nil
}
```

`pgx.ErrNoRows` → `model.ErrNotFound` mapping is correctly handled here. Other pgx errors
are wrapped with context.

Type conversion helpers:
```go
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
    return pgtype.UUID{Bytes: id, Valid: true}
}
func pgtypeToUUID(id pgtype.UUID) uuid.UUID {
    return uuid.UUID(id.Bytes)
}
```

`toOrganization` (lines 81-90) maps `db.Organization` (sqlc-generated pgtype fields) to
`model.Organization` (native Go types / uuid.UUID / time.Time).

---

## 8. sqlc Query Layer

**File:** `packages/shared-domain/queries/organizations.sql`

```sql
-- name: GetOrganizationByID :one
SELECT id, type, name, slug, created_at, updated_at
FROM organizations
WHERE id = @id;
```

No `org_id` filter — correct, because `organizations` is a top-level resource with no
`org_id` column. Access is controlled at the handler layer via `OrgIDsFromContext`.

**Generated Go:** `packages/shared-domain/db/organizations.sql.go:20-32`

```go
func (q *Queries) GetOrganizationByID(ctx context.Context, id pgtype.UUID) (Organization, error) {
    row := q.db.QueryRow(ctx, getOrganizationByID, id)
    var i Organization
    err := row.Scan(&i.ID, &i.Type, &i.Name, &i.Slug, &i.CreatedAt, &i.UpdatedAt)
    return i, err
}
```

`q.db` is the `DBTX` interface (wraps `pgxpool.Pool`). The OTel context carrying the active
span is passed through here, enabling DB-level tracing.

**ListOrganizationsByIDs** uses `WHERE id = ANY($1::uuid[])` — a single query with the
full array of org IDs from JWT claims. No N+1 pattern.

---

## 9. Response Helpers

**File:** `services/api-dashboard/internal/httputil/response.go`

```go
type Response struct {
    Data  any        `json:"data,omitempty"`
    Error *ErrorBody `json:"error,omitempty"`
}

func RespondJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(Response{Data: data})
}

func RespondError(w http.ResponseWriter, status int, code, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(Response{Error: &ErrorBody{Code: code, Message: message}})
}
```

Single source of truth in `httputil` package. Both middleware (`auth.go`) and handlers
(`organization.go`, `health.go`) import from here — no duplicate response logic.

Envelope format: `{"data": ...}` on success, `{"error": {"code": "...", "message": "..."}}` on error.

---

## 10. Complete Call Chain: GET /api/v1/organizations/{id}

```
HTTP Request
  Authorization: Bearer <jwt>
  X-Org-ID: <uuid>

  ↓ otelchi.Middleware        — starts OTel root span
  ↓ chimiddleware.RequestID   — injects X-Request-ID
  ↓ chimiddleware.RealIP      — sets real client IP
  ↓ chimiddleware.Recoverer   — panic safety

  ↓ auth.ValidateJWT          [middleware/auth.go:40]
      • strips "Bearer " prefix
      • jwt.ParseWithClaims → validates HMAC signature, expiry
      • parses claims.Orgs []OrgClaim from JWT body
      • reads X-Org-ID header → uuid.Parse
      • iterates claims.Orgs: builds orgIDs[], finds role for active org
      • if role == "" → 403
      • context.WithValue: org_id, role, org_ids

  ↓ auth.RequireRole("viewer","editor","admin","owner")  [middleware/auth.go:97]
      • RoleFromContext(ctx) → compare against allowed list
      • if no match → 403

  ↓ OrganizationHandler.GetByID  [handler/organization.go:42]
      • chi.URLParam(r, "id") → uuid.Parse
      • OrgIDsFromContext(ctx) → []uuid.UUID
      • slices.Contains(orgIDs, id) → if false → 403
      • h.service.GetByID(ctx, id)

  ↓ OrganizationService.GetByID  [service/organization.go:70]
      • s.tracer.Start(ctx, "OrganizationService.GetByID") → child OTel span
      • slog.InfoContext
      • s.repo.GetByID(ctx, id)

  ↓ organizationRepo.GetByID  [repository/organization.go:46]
      • uuidToPgtype(id) → pgtype.UUID
      • r.q.GetOrganizationByID(ctx, pgtype.UUID)

  ↓ db.Queries.GetOrganizationByID  [packages/shared-domain/db/organizations.sql.go:20]
      • q.db.QueryRow(ctx, "SELECT ... FROM organizations WHERE id = $1", id)
      • row.Scan → db.Organization{pgtype fields}
      • if pgx.ErrNoRows → (bubbles up as model.ErrNotFound)

  ↑ toOrganization(row) → model.Organization{uuid.UUID, time.Time fields}
  ↑ service returns *model.Organization
  ↑ handler: httputil.RespondJSON(w, 200, org)
      → {"data": {"id": "...", "type": "...", ...}}
```

---

## 11. Findings and Noteworthy Observations

### Finding 1: HMAC secret instead of JWKS (architecture deviation)

**Location:** `middleware/auth.go:51`, `config/config.go:17`

The architecture rules state JWT must be validated against BetterAuth JWKS endpoint using
`golang-jwt/jwt/v5`. The actual code uses a symmetric HMAC secret from `JWT_SECRET` env var.
There is no signing method check (`token.Method.(*jwt.SigningMethodHMAC)` guard), which
technically allows algorithm confusion — a token signed with `none` algorithm could bypass
validation depending on library behavior.

### Finding 2: No org_id passed to service in GetByID

**Location:** `handler/organization.go:56`

```go
org, err := h.service.GetByID(r.Context(), id)
```

The active `org_id` (from context) is not passed to the service. The membership check
is done at handler level (`slices.Contains(orgIDs, id)`), before the service call.
For the `organizations` table this is architecturally correct (no `org_id` column on table).
However, the pattern diverges from the canonical sub-resource pattern where `orgID` flows
all the way to SQL.

### Finding 3: Create does not insert org membership

**Location:** `handler/organization.go:33`, `service/organization.go:62`

`POST /api/v1/organizations` creates an org row but does not insert a row into
`org_memberships`. After creation, the calling user has no JWT-visible membership to the
new org (until the JWT is re-issued). No `org_memberships` table or repository exists in
the current codebase.

### Finding 4: RequireRole allows viewer on POST

**Location:** `router/router.go:32`

```go
r.Use(auth.RequireRole("viewer", "editor", "admin", "owner"))
```

All three routes (`POST /`, `GET /`, `GET /{id}`) share the same `RequireRole` middleware
registered at the route group level. This means `viewer`-role users can call
`POST /api/v1/organizations` (create). The architecture rules require mutation endpoints
to have explicit role gates excluding viewer.

### Finding 5: slog structured logging — correct pattern used

The service uses `slog.InfoContext` with typed attributes throughout. Context is propagated
correctly so structured log entries include OTel trace ID correlation.

---

## File Index

| File | Purpose |
|------|---------|
| `services/api-dashboard/cmd/server/main.go` | Entry point, DI wiring |
| `services/api-dashboard/internal/config/config.go` | JWT_SECRET, DATABASE_URL loading |
| `services/api-dashboard/internal/router/router.go` | Chi router, middleware chain |
| `services/api-dashboard/internal/middleware/auth.go` | ValidateJWT, RequireRole, context accessors |
| `services/api-dashboard/internal/handler/organization.go` | HTTP handlers, error mapping |
| `services/api-dashboard/internal/httputil/response.go` | RespondJSON, RespondError |
| `services/api-dashboard/internal/service/organization.go` | Business logic, OTel spans, slog |
| `services/api-dashboard/internal/repository/organization.go` | DB access, pgx error mapping |
| `services/api-dashboard/internal/model/organization.go` | Domain type, sentinel errors |
| `packages/shared-domain/queries/organizations.sql` | sqlc source queries |
| `packages/shared-domain/db/organizations.sql.go` | sqlc generated Go code |
| `packages/shared-domain/db/db.go` | DBTX interface, Queries struct |
| `packages/shared-domain/sqlc.yaml` | sqlc configuration |
