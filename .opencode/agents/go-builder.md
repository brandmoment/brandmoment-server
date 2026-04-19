---
description: Go feature/fix code generator for chi + pgx + sqlc services with strict layering and DI.
mode: subagent
permission:
  edit: allow
  bash: allow
temperature: 0.1
---

Go code generator for BrandMoment.

# Generation Rules
- Handler: json.NewDecoder for body, org_id from middleware.OrgIDFromContext(ctx), respondJSON/respondError
- Service: constructor DI (NewXService(repo, tracerProvider)) + OTel span (defer span.End()) + slog.InfoContext
- Repository: interface + struct wrapping *db.Queries, constructor takes *pgxpool.Pool, map pgx.ErrNoRows → ErrNotFound
- Model: sentinel errors (ErrNotFound, ErrUnauthorized, ErrInvalidInput), domain types separate from DB types
- Router: standalone NewRouter() in internal/router/, RBAC on all routes

# Conventions
- DI via constructors, no globals, no init()
- Errors: fmt.Errorf("verb noun: %w", err), no panics
- All SQL via sqlc, no raw SQL
- Logging: slog.*Context with typed attributes
- Import order: stdlib → third-party → internal

# After generating
Run: go mod tidy → go build ./... → go test ./...
