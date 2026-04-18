# BrandMoment Server

Multi-tenant ad network platform. Monorepo: Go backend + Next.js 15 frontend.

## Tech Stack

| Layer         | Tech                                                                           |
|---------------|--------------------------------------------------------------------------------|
| Backend       | Go 1.23, chi router, pgx, sqlc                                                 |
| Frontend      | Next.js 15 App Router, React 19, TypeScript                                    |
| UI            | shadcn/ui, Tailwind v4                                                         |
| Auth          | BetterAuth (self-hosted in Next.js)                                            |
| DB            | Postgres 17 (OLTP), S3/R2 Parquet (analytics)                                  |
| Analytics     | Rill Developer (internal BI), Recharts (user-facing custom charts)             |
| Observability | OpenTelemetry, Jaeger (dev), Grafana Cloud (prod)                              |
| Migrations    | golang-migrate                                                                 |
| Codegen       | sqlc (Go DB queries), oapi-codegen (Go server), openapi-typescript (TS client) |
| Monorepo      | Turborepo, pnpm 9.15, go.work                                                  |

## Architecture

```
brandmoment-server/
├── services/
│   ├── api-dashboard/       # Go REST API (chi, CRUD, auth middleware)
│   └── api-sdk/             # Go hot-path API for mobile SDKs (v2)
├── apps/
│   └── dashboard/           # Next.js 15 UI (BetterAuth, Rill embed)
├── packages/
│   ├── shared-domain/       # Go shared models, DB queries (sqlc)
│   └── proto/               # OpenAPI spec (source of truth)
├── infra/
│   ├── docker/              # docker-compose (Postgres, MinIO, Rill, OTel, Jaeger)
│   ├── rill/                # Rill dashboards, models, sources
│   ├── seed/                # Go seed data generator
│   └── migrations/          # SQL migrations (golang-migrate)
└── docs/                    # Submodule: external docs repo
```

## Pre-flight Checks

Before generating code, verify tools: `go version`, `sqlc version`, `migrate -version`, `docker --version`, `pnpm --version`, `node --version`. If any missing — stop and report with install commands.

## Post-generation Checks

For ad-hoc requests (outside profiles): `go build ./...` → `go vet ./...` → `go test ./...` → `/simplify` → `/security-review`. Within profiles — validation handled by agents.

## New Entity Checklist

Order: migration → sqlc queries → model → repository → service → handler → tests → router → post-generation checks.

## Multi-Tenancy Model

3 org types: **admin**, **publisher**, **brand**. JWT carries `orgs` array with `{org_id, role}`. Roles: `owner | admin | editor | viewer`.

**org_id filtering**: sub-resources (campaigns, publisher_apps, api_keys, rules) ALWAYS `WHERE org_id = $1`. Organizations table has NO org_id — access via JWT membership. Users accessed through `org_memberships` join. Do NOT blindly add org_id to every query.

## Auth: BetterAuth

Go services validate JWTs via `golang-jwt/jwt/v5` against BetterAuth JWKS. No custom JWT parsing. RBAC middleware checks roles from `orgs` array in JWT claims.

## Naming Conventions

**Go**: packages lowercase no underscores, files `snake_case.go`, types PascalCase with suffix, constructors `New*`, errors `Err*`, tests `Test*_Method`.
**TypeScript**: components PascalCase, hooks `use*`, types PascalCase no `I` prefix, constants `SCREAMING_SNAKE`.
**Database**: tables `snake_case` plural, PKs `id UUID`, FKs `{entity}_id`, timestamps `created_at`/`updated_at`, booleans `is_*`, enums TEXT+CHECK.
**API**: `kebab-case` endpoints, versioned `/api/v1/...`, envelope `{"data": ..., "error": {"code": "...", "message": "..."}}`.

## Dev Commands

```bash
make infra-up / infra-down    # Docker stack (Postgres, MinIO, Rill, OTel, Jaeger)
make seed                     # 50k session events → Parquet → MinIO
make rill-ui / jaeger-ui / minio-ui  # Open UIs
```

## Tools & Skills

- **RTK**: token-optimized CLI proxy, transparent via hooks. See `~/.claude/RTK.md`
- **ast-index**: `/ast-index` for symbol search, usages, implementations, project map. Prefer over Grep/Glob for symbols
- **Plugins**: `gopls-lsp` (enabled), `typescript-lsp` / `frontend-design` / `playwright` (disabled). Enable: `claude plugin enable <name>`
- **JetBrains MCP**: `build_project`, `get_file_problems`, `search_symbol`, `execute_run_configuration`
- **Post-generation skills**: `/simplify`, `/security-review`, `/review`, `/frontend-design`

---

## Profile Selection

Each request → ONE profile, auto-detected by keywords:
- Bug/error/crash/500/regression/broken/not working → **Bug Fix**
- How/what/where/why/explain/research/investigate → **Research**
- Docs/document/sync docs → **Update Docs**
- Add/create/implement/build/new/update/change/modify/refactor → **Feature**
- Run tests/validate/verify/check → **Verification**

High confidence → proceed immediately with `[Profile: <name>]`. Ambiguous → confirm via `AskUserQuestion`. User can specify explicitly.

**Pre-flight**: scan `reports/*/_status.md` for non-Done tasks matching context. If found — show stage, offer to continue. Never restart without asking.

**Create workspace** (if no match): `mkdir -p reports/<slug>/` + write `_status.md`.

### Available Agents

Defined in `.claude/agents/`. Launched via `Agent` tool.

| Category | Agents |
|----------|--------|
| Builders (write code) | `go-builder`, `ts-builder`, `sql-builder`, `go-test-writer`, `e2e-test-writer`, `refactor-go` |
| Experts (read-only) | `go-diagnostics`, `ts-diagnostics`, `sql-analyzer`, `rill-analyzer`, `security-reviewer`, `git-investigator`, `docs-analyzer` |
| Utility | `test-runner` (sonnet), `report-writer` (haiku), `system-analytics` (sonnet) |

---

## Agent & Workspace Rules (all profiles)

### Delegation

Main orchestrates; agents execute. Main NEVER writes source code, runs the validation suite, reads code for diagnosis, or writes reports directly. Main MAY run single repro commands (`curl`, `go test -run TestSpecific`) at Reproduce stage. Main MAY write docs and synthesis files (`02-analyze.md`, `02-plan.md`). Agent launch is mandatory at every stage — no shortcuts even for trivial tasks.

### Validation Checks (used by `test-runner`)

| Stack | Checks |
|-------|--------|
| Go (`services/`, `packages/`) | `go build ./...` → `go vet ./...` → `go test ./...` |
| TypeScript (`apps/dashboard/`) | `pnpm typecheck` → `pnpm lint` → `pnpm test` → `playwright e2e` |
| SQL (`infra/migrations/`) | `sqlc generate` |
| Auth/RBAC changes | `/security-review` |

### Validation Failure Routing

- Build/vet/lint fails → back to build stage (source broken)
- Test assertion fails → back to build stage (logic wrong)
- Test compilation error → back to test-writing stage
- E2E element/timeout → back to build or test stage depending on cause
- 2+ fix iterations fail → back to diagnose stage (root cause was wrong)

Loop until green. 3 iterations without progress → escalate to user.

### Workspace

Each task: `reports/<slug>/` with `_status.md` replaced at each transition. Slug: `<description>-<profile>`.

Files named `NN-stage-agent.md` (e.g., `02-diagnose-go.md`). Parallel agents write separate files.

```markdown
# _status.md format
# Task: <title>
Profile: <profile>  |  Stage: <current>  |  Next: <agent>
Created: YYYY-MM-DD  |  Updated: YYYY-MM-DD HH:MM

## Context
<brief description>

## Handoff
next: <agent> | reason: <why> | input: <file paths, root cause, etc.>
```

**Rules**:
1. Main creates workspace + `_status.md` at start
2. Agents read previous stage files for context (main does NOT pre-read or rephrase)
3. Main passes workspace path + output filename in each agent prompt
4. Loop iterations overwrite (no `-v2` files)
5. Empty agent result → note "no findings" and proceed
6. Launch agents first — bookkeeping can wait
7. Background agents: wait for notification before reading output. No polling

---

## Profile: Bug Fix

```
Reproduce → Diagnose → Fix → Test → Validate → Report → Done
```

**Transitions**: Reproduce→Diagnose, Reproduce→Report (not reproducible), Diagnose→Fix, Diagnose→Reproduce, Diagnose→Report (diagnosis only), Fix→Test, Fix→Diagnose, Test→Validate, Validate→Report/Fix/Test/Diagnose, Report→Done. Log `[Stage: X → Y]`.

| Stage | Agents | Main's job |
|-------|--------|------------|
| Reproduce | main | Run tests/curl, write `01-reproduce.md` with bug description + repro steps + stack. 3 failed attempts → ask user or Report |
| Diagnose | (`go-diagnostics` or `ts-diagnostics`) + `git-investigator` + `security-reviewer` | Launch parallel. If agents agree on root cause → Fix. If disagree → escalate to user |
| Fix | `go-builder` / `ts-builder` / `sql-builder` | Launch builder. If fix touches auth → flag for security review. Fix the code, not the test |
| Test | `go-test-writer` / `e2e-test-writer` | Regression test must fail if fix is reverted |
| Validate | `test-runner` | Route failures per validation rules above |
| Report | `report-writer` | Compiles report, sets `Stage: Done` |

---

## Profile: Feature

```
Spec → Implement → Test → Validate → Report → Done
```

**Transitions**: Spec→Implement, Implement→Test/Spec, Test→Validate/Implement, Validate→Report/Implement/Test, Report→Done. Log `[Stage: X → Y]`.

| Stage | Agents | Main's job |
|-------|--------|------------|
| Spec | main / `system-analytics` | Write `01-spec.md`: what to build, layers, acceptance criteria |
| Implement | `go-builder` + `sql-builder` + `ts-builder` (parallel by stack). For refactoring: `refactor-go` instead of `go-builder` | SQL first if SQL+Go/TS, then others parallel |
| Test | `go-test-writer` + `e2e-test-writer` | Table-driven unit tests + smoke scenarios |
| Validate | `test-runner` | Route failures per validation rules |
| Report | `report-writer` | Compiles report, sets `Stage: Done` |

Exception: SQL-only changes may skip Test with `[Skip Test: SQL-only]`.

---

## Profile: Research

```
Explore → Analyze → Report → Done
```

**Transitions**: Explore→Analyze/Report (only if agents confirm trivial), Analyze→Explore/Report, Report→Done.

| Stage | Agents | Main's job |
|-------|--------|------------|
| Explore | At least 2 of: `go-diagnostics`, `ts-diagnostics`, `sql-analyzer`, `rill-analyzer`, `docs-analyzer` | Select by topic, launch parallel |
| Analyze | main | Synthesize findings. Classify: CRITICAL / IMPORTANT / NOTE. Include "Not investigated" section |
| Report | `report-writer` | Compiles report, sets `Stage: Done` |

**Research MUST NOT modify any code files.**

---

## Profile: Update Docs

```
Analyze Code → Plan Changes → Update Docs → Validate → Report → Done
```

**Transitions**: AnalyzeCode→Plan, Plan→Update, Update→Validate/AnalyzeCode, Validate→Report/Update, Report→Done.

| Stage | Agents | Main's job |
|-------|--------|------------|
| Analyze Code | `docs-analyzer` + `go-diagnostics` + `ts-diagnostics` | Launch parallel |
| Plan Changes | main | List sections to create/update. Priority: TODOs > stale > new |
| Update Docs | main | Write docs (Russian content, English headers). Map: `docs/backend/` ↔ `services/`, `docs/dashboard/` ↔ `apps/dashboard/` |
| Validate | `docs-analyzer` | Check links, orphan pages, README navigation |
| Report | `report-writer` | Compiles report, sets `Stage: Done` |

**Update Docs MUST NOT modify source code.** Only `docs/` files.

---

## Profile: Verification

```
Scan → Update Smoke → Run → Report → Done
```

**Transitions**: Scan→UpdateSmoke (new UI features) or Scan→Run (no new features), UpdateSmoke→Run, Run→Report, Report→Done.

| Stage | Agents | Main's job |
|-------|--------|------------|
| Scan | (`go-diagnostics` or `ts-diagnostics`) + `git-investigator` | Detect what changed. New UI features → Update Smoke, else → Run |
| Update Smoke | `e2e-test-writer` | Generate scenarios for changed features |
| Run | `test-runner` | Run ALL checks. Report failures with file:line + suggested location |
| Report | `report-writer` | Unified report: scanned files, new scenarios, results table, failures, screenshots |

**Verification MUST NOT modify source code or fix failures.** Report-only.

---

## Detailed Rules

Go patterns, anti-patterns, and code examples are in `.claude/rules/` — loaded automatically by file glob.
