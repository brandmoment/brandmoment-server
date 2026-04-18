# Security Review: JWT Authorization Flow

Agent: security-reviewer
Date: 2026-04-18

## Scope

Files reviewed:
- `/services/api-dashboard/internal/middleware/auth.go`
- `/services/api-dashboard/internal/router/router.go`
- `/services/api-dashboard/internal/handler/organization.go`
- `/services/api-dashboard/internal/service/organization.go`
- `/services/api-dashboard/internal/repository/organization.go`
- `/services/api-dashboard/internal/config/config.go`
- `/services/api-dashboard/cmd/server/main.go`
- `/packages/shared-domain/db/organizations.sql.go`
- `/packages/shared-domain/queries/organizations.sql`

---

## CRITICAL

### FINDING-01: JWT validated with symmetric HMAC secret instead of JWKS (A02, A07)

**Location**: `/services/api-dashboard/internal/middleware/auth.go:51-56`

**Description**: The rule file (`go-multi-tenancy.md`) and project CLAUDE.md both explicitly mandate that JWTs must be validated against BetterAuth's JWKS endpoint using `golang-jwt/jwt/v5`. Instead, `NewAuth` accepts a raw `jwtSecret []byte` (symmetric HMAC key) and `ParseWithClaims` passes `a.jwtSecret` as the key material. This means:

1. The algorithm is implicitly trusted from the token header — the key function returns the symmetric secret without checking that `token.Method` is `*jwt.SigningMethodHMAC`. An attacker who controls a token can set `alg: none` or `alg: HS256` with a known/guessable secret to forge tokens.
2. BetterAuth issues asymmetric JWTs (RS256/ES256 via JWKS). A symmetric secret here means the server is NOT validating tokens issued by BetterAuth — it will accept any token signed with whatever `JWT_SECRET` is configured, which is completely decoupled from BetterAuth's key material.

**OWASP**: A02 (Cryptographic Failures), A07 (Identification and Authentication Failures)

**Proof**:
```go
// middleware/auth.go:36-38
func NewAuth(jwtSecret string) *Auth {
    return &Auth{jwtSecret: []byte(jwtSecret)}
}

// middleware/auth.go:51-56
token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
    return a.jwtSecret, nil  // NO algorithm check, NO JWKS fetch
})
```

**Fix**: Fetch the JWKS from BetterAuth at startup (or on first use with caching), validate `token.Method` is `*jwt.SigningMethodRSA` or `*jwt.SigningMethodECDSA`, and return the matching public key. Use `github.com/MicahParks/keyfunc` or equivalent JWKS-aware key function. Never accept HMAC tokens from BetterAuth.

---

### FINDING-02: No algorithm validation in JWT key function — "alg:none" bypass possible (A02, A07)

**Location**: `/services/api-dashboard/internal/middleware/auth.go:51-56`

**Description**: The `jwt.ParseWithClaims` key function unconditionally returns `a.jwtSecret` without checking `token.Method`. While `golang-jwt/jwt/v5` rejects `alg:none` by default when a non-nil key is returned, the absence of an explicit algorithm allowlist means that if the library's behaviour changes or a future developer switches to a different JWT library, this silently accepts any algorithm. This is a defence-in-depth gap directly violating the project's explicit rule "validate against BetterAuth JWKS".

**OWASP**: A02

**Proof**: Same snippet as FINDING-01 — no `if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok { return nil, errors.New("unexpected signing method") }` guard exists.

**Fix**: Add explicit algorithm check inside the key function:
```go
func(token *jwt.Token) (any, error) {
    if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
    }
    return jwksPublicKey(token), nil
}
```

---

### FINDING-03: `POST /api/v1/organizations` — any authenticated viewer can create organizations (A01, Broken Access Control)

**Location**: `/services/api-dashboard/internal/router/router.go:31-36`

**Description**: The `/organizations` route group applies `RequireRole("viewer", "editor", "admin", "owner")` to ALL methods including `POST /`. This means a `viewer`-role user can create new organizations. Creating an organization is a privileged mutation and must require at minimum `editor` or `owner`. The `viewer` role is documented as read-only.

**OWASP**: A01 (Broken Access Control)

**Proof**:
```go
// router/router.go:31-36
r.Route("/organizations", func(r chi.Router) {
    r.Use(auth.RequireRole("viewer", "editor", "admin", "owner"))  // viewer allowed on POST
    r.Post("/", h.Organization.Create)
    r.Get("/", h.Organization.List)
    r.Get("/{id}", h.Organization.GetByID)
})
```

**Fix**: Apply `RequireRole` per-verb, not per-group, or split the route group:
```go
r.Get("/", h.Organization.List)
r.Get("/{id}", h.Organization.GetByID)
r.With(auth.RequireRole("owner")).Post("/", h.Organization.Create)
```

---

### FINDING-04: No issuer (`iss`) or audience (`aud`) validation on JWT (A07)

**Location**: `/services/api-dashboard/internal/middleware/auth.go:51-56`

**Description**: `jwt.ParseWithClaims` is called without `jwt.WithIssuer(...)` or `jwt.WithAudience(...)` parser options. Any token signed with the same HMAC secret but issued by a different service (or an attacker) will be accepted. In a multi-service environment this is a cross-service token replay vector.

**OWASP**: A07

**Proof**:
```go
token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
    return a.jwtSecret, nil
})
// No jwt.WithIssuer / jwt.WithAudience options passed
```

**Fix**:
```go
token, err := jwt.ParseWithClaims(tokenStr, claims, keyFunc,
    jwt.WithIssuer("https://auth.brandmoment.io"),
    jwt.WithAudience("api-dashboard"),
    jwt.WithExpirationRequired(),
)
```

---

## HIGH

### FINDING-05: JWT expiration not explicitly required (A07)

**Location**: `/services/api-dashboard/internal/middleware/auth.go:51-56`

**Description**: `jwt.ParseWithClaims` in `golang-jwt/jwt/v5` validates `exp` if the claim is present, but does NOT reject tokens that have no `exp` claim at all unless `jwt.WithExpirationRequired()` is passed. BetterAuth tokens should always carry `exp`, but there is no defence if a token is crafted without it — it would never expire.

**OWASP**: A07

**Fix**: Pass `jwt.WithExpirationRequired()` as a parser option.

---

### FINDING-06: Empty `orgIDs` slice leads to `ListOrganizationsByIDs` returning zero rows silently — but is not a security issue by itself; however the SQL query accepts a client-controlled array size with no upper bound (A04)

**Location**: `/services/api-dashboard/internal/repository/organization.go:57-70`, `/packages/shared-domain/db/organizations.sql.go:77`

**Description**: `ListOrganizationsByIDs` passes a `[]pgtype.UUID` of arbitrary length directly to `WHERE id = ANY($1::uuid[])`. The slice size is bounded by the JWT's `orgs` array, but there is no server-side limit on how many org IDs a JWT can carry. A token with thousands of org IDs would cause a large `IN`-equivalent query and is a mild denial-of-service / resource exhaustion vector.

**OWASP**: A04 (Insecure Design)

**Fix**: Add a cap: `if len(ids) > 100 { ids = ids[:100] }` or validate JWT `orgs` array length at parse time in middleware.

---

### FINDING-07: `handleServiceError` leaks sentinel error messages directly to the HTTP response (A09, Information Disclosure)

**Location**: `/services/api-dashboard/internal/handler/organization.go:77-88`

**Description**: For `ErrNotFound` and `ErrInvalidInput`, `err.Error()` is passed verbatim as the HTTP error message:
```go
case errors.Is(err, model.ErrNotFound):
    httputil.RespondError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
```
`model.ErrNotFound` is defined as `errors.New("not found")` — safe on its own. But errors wrapped with additional context (e.g., `fmt.Errorf("get organization: %w", ErrNotFound)`) would expose the wrapping message including internal operation names. If a service wraps with context before returning, this leaks internals.

More critically: `model.ErrUnauthorized` maps to HTTP 401 with `err.Error()` as message, yet the middleware already sends 403 for org membership failures — if a service layer returns `ErrUnauthorized` with a descriptive message this could leak internal auth logic details.

**OWASP**: A09

**Fix**: Use static strings for HTTP error messages; never forward `err.Error()` to clients. Log the full error server-side.

---

## MEDIUM

### FINDING-08: No rate limiting on any endpoint (A04, A05)

**Location**: `/services/api-dashboard/internal/router/router.go`

**Description**: No rate-limiting middleware is applied anywhere in the router — not on auth-adjacent endpoints nor on the organization mutation endpoint. This enables credential brute-force (if JWT secrets are short/guessable) and resource exhaustion.

**OWASP**: A04, A05

**Fix**: Add `golang.org/x/time/rate` or a chi-compatible rate-limiter middleware (e.g., `github.com/go-chi/httprate`) at the top of the router for at minimum the `/api/v1` route group.

---

### FINDING-09: No request body size limit (A05)

**Location**: `/services/api-dashboard/cmd/server/main.go:66-73`

**Description**: The `http.Server` has no `MaxHeaderBytes` override and no `http.MaxBytesReader` wrapper in handlers. A client can send an arbitrarily large JSON body to `POST /api/v1/organizations` causing memory exhaustion.

**OWASP**: A05

**Fix**: Wrap `r.Body` in handler before decode:
```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
```
Or add a chi middleware globally: `r.Use(chimiddleware.RequestSize(1 << 20))`.

---

### FINDING-10: No CORS configuration (A05)

**Location**: `/services/api-dashboard/internal/router/router.go`

**Description**: No CORS middleware is configured. For a dashboard API this means browsers rely on the default same-origin policy without explicit CORS headers, which may be overly permissive depending on how the frontend calls the API (if the Next.js dashboard is on a different origin). If the API is ever called cross-origin without CORS headers, preflight requests will fail silently or be bypassed by non-browser clients.

**OWASP**: A05

**Fix**: Add `github.com/go-chi/cors` middleware with explicit allowed origins.

---

### FINDING-11: OTel exporter uses insecure gRPC (A02)

**Location**: `/services/api-dashboard/cmd/server/main.go:97-100`

**Description**: `otlptracegrpc.WithInsecure()` is hardcoded — traces are sent over plaintext gRPC. In production, this means trace data (which includes org_id, operation names, potentially request metadata) is transmitted unencrypted.

**OWASP**: A02

**Fix**: Make TLS for OTLP configurable via env var; default to TLS in production builds.

---

### FINDING-12: Hardcoded MinIO credentials in seed tool (A02)

**Location**: `/infra/seed/main.go:133-137`

**Description**: MinIO credentials `minioadmin` / `minioadmin` are hardcoded in the seed binary. While this is a dev-only tool, if the seed binary is ever run against a staging/prod MinIO instance (or if the default credentials are never rotated), this is a credential exposure.

**OWASP**: A02

**Fix**: Read credentials from environment variables; document that these must be changed before staging deployment.

---

## LOW

### FINDING-13: `X-Org-ID` membership check skips UUIDs that fail to parse (A01)

**Location**: `/services/api-dashboard/internal/middleware/auth.go:73-81`

**Description**: When building `orgIDs` from JWT claims, UUIDs that fail `uuid.Parse` are silently skipped with `continue`. If the JWT contains a malformed org_id string that happens to match `xOrgID` as a string but not as a UUID, the membership check will never find a role match and the request is correctly rejected. However, the silent skip means malformed claims are never logged, making it impossible to detect JWT tampering or BetterAuth bugs that produce invalid UUIDs.

**OWASP**: A09

**Fix**: Log a warning for each unparseable org_id claim.

---

### FINDING-14: Context key type collision risk (A04)

**Location**: `/services/api-dashboard/internal/middleware/auth.go:14-19`

**Description**: `contextKey` is a private type within the `middleware` package. This is correct — it prevents collisions with other packages using plain string keys. However, the three context keys (`ctxOrgID`, `ctxRole`, `ctxOrgIDs`) are all of type `contextKey` with string underlying values. If a future developer adds a new package with a different `contextKey` type, the accessor functions `OrgIDFromContext` / `RoleFromContext` return zero values silently (the type assertion `.(uuid.UUID)` will fail and return `uuid.Nil`). There is no nil/zero-value guard in the accessor functions.

**OWASP**: A04

**Fix**: Add a guard that panics or returns an error if the context value is missing:
```go
func OrgIDFromContext(ctx context.Context) uuid.UUID {
    id, ok := ctx.Value(ctxOrgID).(uuid.UUID)
    if !ok || id == uuid.Nil {
        panic("OrgIDFromContext: org_id not set in context — ValidateJWT middleware missing")
    }
    return id
}
```

---

### FINDING-15: No `Content-Type: application/json` validation on incoming requests (A05)

**Location**: `/services/api-dashboard/internal/handler/organization.go:27-31`

**Description**: `json.NewDecoder(r.Body).Decode(&req)` is called without first checking that `Content-Type` is `application/json`. This is a minor issue since `json.Decoder` will attempt to parse any body, but is a best-practice gap.

**OWASP**: A05

**Fix**: Check `r.Header.Get("Content-Type") == "application/json"` before decoding, or add a global chi middleware.

---

## Summary Table

| ID  | Severity | Title                                                          | OWASP |
|-----|----------|----------------------------------------------------------------|-------|
| 01  | CRITICAL | HMAC secret instead of JWKS — BetterAuth tokens not validated | A02, A07 |
| 02  | CRITICAL | No algorithm check in key function                             | A02 |
| 03  | CRITICAL | Viewer role can POST (create) organizations                    | A01 |
| 04  | CRITICAL | No issuer/audience validation on JWT                           | A07 |
| 05  | HIGH     | JWT expiration not explicitly required                         | A07 |
| 06  | HIGH     | Unbounded org ID array in ListByIDs query                      | A04 |
| 07  | HIGH     | Service error messages leaked verbatim in HTTP responses       | A09 |
| 08  | MEDIUM   | No rate limiting on any endpoint                               | A04, A05 |
| 09  | MEDIUM   | No request body size limit                                     | A05 |
| 10  | MEDIUM   | No CORS configuration                                          | A05 |
| 11  | MEDIUM   | OTel exporter uses plaintext gRPC (WithInsecure hardcoded)    | A02 |
| 12  | MEDIUM   | Hardcoded MinIO credentials in seed tool                       | A02 |
| 13  | LOW      | Silent skip of unparseable JWT org_id claims                   | A09 |
| 14  | LOW      | Context accessor functions return zero value silently          | A04 |
| 15  | LOW      | No Content-Type validation on incoming requests                | A05 |

---

## What Is Correct (positive findings)

- X-Org-ID is validated against JWT `orgs` array — membership is checked before storing org_id in context (`auth.go:73-86`).
- `org_id` is never taken from request body — always from JWT context (`handler/organization.go:50, 66`).
- `organizations` table correctly has no `org_id` column (it IS the top-level resource); sub-resource filtering would be needed when sub-resources are added.
- `ListByIDs` and `GetByID` in the handler correctly use the org IDs from context, not from request params.
- `GetByID` performs an explicit membership check before calling the service (`handler/organization.go:51-54`).
- sqlc is used for all DB queries — no raw SQL injection vectors.
- `model.ErrInvalidInput` / `ErrNotFound` / `ErrUnauthorized` are sentinel errors, not wrapping internal DB errors directly to clients (though see FINDING-07 for the partial leak).
- `golang-jwt/jwt/v5` library is used (not manual parsing) — partially satisfies the rule, but with wrong key material (see FINDING-01).
