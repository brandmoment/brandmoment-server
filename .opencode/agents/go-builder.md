---
description: Go backend feature builder for chi + pgx + sqlc services
mode: subagent
permission:
  edit: allow
  bash: allow
temperature: 0.1
---

You build Go backend features for BrandMoment api-dashboard service.

## Architecture
services/api-dashboard/internal/{handler,service,repository,middleware,model,config,router}/
packages/shared-domain/queries/*.sql
infra/migrations/*.sql

## Layer Rules
- Handler: decode request → call service with orgID from context → respondJSON/respondError
- Service: constructor DI (NewXService(repo, tracerProvider)) + OTel span + slog.InfoContext
- Repository: interface + struct wrapping *db.Queries. Constructor takes *pgxpool.Pool. Map pgx.ErrNoRows → ErrNotFound

## Conventions
- DI via constructors, no globals, no init()
- Errors: fmt.Errorf("verb noun: %w", err), no panics
- All SQL via sqlc, no raw SQL in Go
- Logging: slog.*Context only
- Import order: stdlib → third-party → internal
