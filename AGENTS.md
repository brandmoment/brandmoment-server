# BrandMoment Server

Multi-tenant ad network platform. Monorepo: Go 1.23 backend + Next.js 15 frontend.

## Tech Stack
- Backend: Go 1.23, chi router, pgx, sqlc
- Frontend: Next.js 15, React 19, TypeScript, shadcn/ui, Tailwind v4
- DB: Postgres 17, migrations via golang-migrate
- Auth: BetterAuth (JWT via golang-jwt/jwt/v5)
- Observability: OpenTelemetry, Jaeger

## Project Structure
```
services/api-dashboard/internal/{handler,service,repository,middleware,model,config,router}/
packages/shared-domain/queries/*.sql
infra/migrations/*.sql
apps/dashboard/ (Next.js)
```

## Go Rules
- Layers: handler (decode/respond) → service (logic + OTel + slog) → repository (wraps sqlc)
- DI via constructors (NewXService), no globals, no init()
- Errors: fmt.Errorf("verb noun: %w", err), no panics
- All SQL via sqlc, no raw SQL in Go
- Logging: slog.*Context with typed attributes
- Tests: table-driven, TestTypeName_Method
- Import order: stdlib → third-party → internal

## Multi-Tenancy
- 3 org types: admin, publisher, brand
- Sub-resources ALWAYS WHERE org_id = $1
- org_id from JWT context, never from request body
- Organizations/users have NO org_id column

## Naming
- Go: snake_case files, PascalCase types, New* constructors, Err* errors
- DB: snake_case plural tables, UUID PKs, org_id FK on sub-resources
- API: kebab-case /api/v1/, envelope {"data","error"}

## SQL
- Migrations: infra/migrations/, up.sql + down.sql, IF NOT EXISTS, TIMESTAMPTZ
- sqlc: Get*ByID, List*sByOrg, Insert*, Update*, Delete*

## New Entity Order
migration → sqlc queries → repository → service → handler → tests → router

## Subagents
- @go-builder — Go backend feature builder
- @sql-builder — migrations + sqlc queries
- @code-reviewer — read-only code review
- @test-writer — table-driven Go tests
