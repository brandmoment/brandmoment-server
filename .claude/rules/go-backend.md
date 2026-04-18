---
description: Go backend patterns, anti-patterns, and code templates for api-dashboard and api-sdk services
globs: "**/*.go"
---

# Go Backend Rules

## Service Structure

```
services/api-dashboard/
├── cmd/server/main.go           # Entry point: config, DI, server start
├── internal/
│   ├── config/config.go         # Env-based config (envconfig)
│   ├── router/router.go         # chi router setup (standalone NewRouter())
│   ├── handler/*.go             # HTTP: decode → delegate → respond
│   ├── service/*.go             # Business logic + OTel span + slog
│   ├── repository/*.go          # Wraps sqlc Queries, maps pgx.ErrNoRows → ErrNotFound
│   ├── middleware/{auth,rbac}.go # JWT validation + role checks
│   └── model/*.go               # Domain types + sentinel errors
```

## Import Order

stdlib → third-party → internal (separated by blank lines).

## Layer Patterns

- **Handler**: decode request → call service with `orgID` from context → respond with `respondJSON`/`respondError`. No business logic
- **Service**: constructor DI (`NewXService(repo, tracerProvider)`) + OTel span (`tracer.Start`) + `slog.InfoContext` with typed attrs + call repo. Return errors with `fmt.Errorf("verb noun: %w", err)`
- **Repository**: interface + struct wrapping `*db.Queries`. Constructor takes `*pgxpool.Pool`. Map `pgx.ErrNoRows` → `ErrNotFound`
- **Router**: standalone `NewRouter(handlers, auth)` in `internal/router/`. Apply middleware: otelchi → RequestID → RealIP → Recoverer → auth → RBAC per route group
- **Response helpers**: `respondJSON(w, status, data)` and `respondError(w, status, code, msg)` — shared across handlers AND middleware. One source of truth

## Anti-Patterns

1. No global state or `init()` — DI via constructors
2. No panics in business logic — return errors
3. No raw SQL anywhere — all SQL in sqlc query files, repository wraps `*db.Queries` only
4. No `fmt.Println` / `log.Println` — use `slog.*Context` with typed attributes
5. No custom JWT parsing — use `golang-jwt/jwt/v5` against BetterAuth JWKS
6. No duplicate response helpers — middleware imports from handler package
7. No guessing dependency versions — `go mod tidy` after creating `go.mod`

## Tests

Table-driven, named `TestTypeName_Method`. Every service method must have tests.

```go
tests := []struct {
    name    string
    input   T
    wantErr bool
}{ /* cases */ }
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { /* ... */ })
}
```
