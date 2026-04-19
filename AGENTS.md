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

---

## Subagents

- @go-builder — Go backend feature builder
- @sql-builder — migrations + sqlc queries
- @code-reviewer — read-only code review
- @test-writer — table-driven Go tests

---

## Profile Selection

Each task → ONE profile, auto-detected by label or keywords:
- `bug` / error / crash / broken → **Bug Fix**
- `test` / test coverage → **Test**
- `refactor` / consolidate / extract → **Refactor**
- `enhancement` / feat / add / create / implement → **Feature**
- `documentation` / docs / godoc → **Docs**

---

## Validation Checks

Run after every code change:

| Stack | Checks |
|-------|--------|
| Go (`services/`, `packages/`) | `go build ./...` → `go vet ./...` → `go test ./...` |
| TypeScript (`apps/dashboard/`) | `pnpm typecheck` → `pnpm lint` → `pnpm test` |
| SQL (`infra/migrations/`) | `sqlc generate` |

### Validation Failure Routing

- Build/vet/lint fails → fix source code
- Test assertion fails → fix logic, NOT the test expectation
- Test compilation error → fix test code
- 3 iterations without progress → stop, report failure

---

## Profile: Bug Fix

```
Reproduce → Diagnose → Fix → Validate → Done
```

1. **Reproduce**: run the failing test or reproduce the bug. If can't reproduce in 3 attempts → skip
2. **Diagnose**: read the relevant code, find root cause
3. **Fix**: apply the fix. Fix the code, not the test
4. **Validate**: run `go build ./...` → `go vet ./...` → `go test ./...`
5. If validation fails → back to Fix (max 3 attempts)

---

## Profile: Feature

```
Read → Implement → Validate → Done
```

1. **Read**: understand what's needed, read existing related code
2. **Implement**: write the code following New Entity Order if applicable
3. **Validate**: run `go build ./...` → `go vet ./...` → `go test ./...`
4. If validation fails → back to Implement (max 3 attempts)

---

## Profile: Refactor

```
Read → Refactor → Validate → Done
```

1. **Read**: understand current code, identify what to change
2. **Refactor**: apply changes. Don't change behavior, only structure
3. **Validate**: run `go build ./...` → `go vet ./...` → `go test ./...`
4. If validation fails → back to Refactor (max 3 attempts)

---

## Profile: Test

```
Read → Write Tests → Validate → Done
```

1. **Read**: understand the code to test
2. **Write Tests**: table-driven, TestTypeName_Method format
3. **Validate**: run `go test ./...` — all tests must pass
4. If validation fails → fix tests (max 3 attempts)

---

## Profile: Docs

```
Read → Write Docs → Done
```

1. **Read**: understand the code that needs documentation
2. **Write Docs**: add godoc comments to exported types, functions, constructors
3. Do NOT modify any code logic — only add/update comments

---

## Execution Loop

Autonomous sequential execution from GitHub issues.

### Protocol

1. Run: `gh issue list --repo brandmoment/brandmoment-server --state open --limit 1 --json number,title,labels,body`
2. Read issue title, labels, and body
3. Map label → profile (see Profile Selection above)
4. Execute per mapped profile (all stage rules apply)
5. Run validation checks
6. Commit with: `<type>: <description>\n\nFixes #N`
7. Run: `gh issue close N --repo brandmoment/brandmoment-server`
8. Go to step 1 — take next issue

### Commit Type Mapping
- `bug` label → `fix:`
- `enhancement` label → `feat:`
- `refactor` label → `refactor:`
- `test` label → `test:`
- `documentation` label → `docs:`

### Stop Conditions
- All issues closed → done
- 3 consecutive failures on same issue → comment on issue, skip to next
- 3 skipped issues in a row → stop, escalate
- Build/test broken and can't self-heal in 3 attempts → stop

### Important Rules
- NEVER skip validation (go build, go vet, go test)
- NEVER modify test expectations to make tests pass — fix the code
- Read existing code before making changes
- Follow all conventions from this file
- Each issue = one commit with `Fixes #N`
- Do NOT ask the user anything — work autonomously
