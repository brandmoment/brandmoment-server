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

Before generating code, verify required tools are installed:

| Tool           | Check command      | Required for                        |
|----------------|--------------------|-------------------------------------|
| Go 1.23+       | `go version`       | Backend services                    |
| sqlc           | `sqlc version`     | DB query codegen                    |
| golang-migrate | `migrate -version` | DB migrations                       |
| Docker         | `docker --version` | Infra (Postgres, MinIO, Rill, OTel) |
| pnpm 9.15+     | `pnpm --version`   | Frontend, monorepo                  |
| Node 20+       | `node --version`   | Frontend                            |

If any tool is missing — **stop and report** which tools need to be installed. Suggest install commands (e.g. `brew install go`). Do NOT proceed with code generation without required tools.

## Post-generation Checks

For ad-hoc requests (outside profiles), verify after generating code:

1. `go build ./...` — compiles without errors
2. `go vet ./...` — no static analysis issues
3. `go test ./...` — all tests pass
4. Run `/simplify` — check for code duplication and quality
5. Run `/security-review` — check multi-tenancy isolation and auth

When working within a profile — validation is handled by agents (see profile workflows).

## New Entity Checklist

When adding a new domain entity (e.g. campaign, publisher-app), follow this order:

1. SQL migration (`infra/migrations/NNNNNN_create_<entity>.up.sql` + `.down.sql`)
2. sqlc queries (`packages/shared-domain/queries/<entity>.sql`)
3. Model (`internal/model/<entity>.go`)
4. Repository (`internal/repository/<entity>.go` — wraps sqlc-generated code)
5. Service (`internal/service/<entity>.go` — business logic + OTel + slog)
6. Handler (`internal/handler/<entity>.go` — HTTP layer)
7. Tests (`internal/service/<entity>_test.go` — table-driven, minimum)
8. Router — add routes in `internal/router/router.go`
9. Run post-generation checks

## Multi-Tenancy Model

3 org types: **admin**, **publisher**, **brand**.

JWT (issued by BetterAuth) carries org memberships:
```json
{
  "sub": "user_uuid",
  "orgs": [
    {"org_id": "uuid", "role": "owner"},
    {"org_id": "uuid", "role": "viewer"}
  ]
}
```

Roles: `owner | admin | editor | viewer`.

### When org_id filtering applies

- **Sub-resources** (campaigns, publisher_apps, api_keys, rules): ALWAYS filter `WHERE org_id = $1`
- **Organizations table itself**: NO org_id column. Access controlled by JWT membership — user sees only orgs they belong to
- **Users**: accessed through `org_memberships` join, not direct org_id on user row

Do NOT blindly add org_id filter to every query. Think about whether the entity is a top-level resource or a sub-resource.

## Auth: BetterAuth

Auth is handled by BetterAuth (self-hosted in Next.js dashboard). Go services validate JWTs issued by BetterAuth.

- Do NOT write custom JWT parsing (no manual HMAC, no manual base64)
- Use a JWT library (`golang-jwt/jwt/v5`) to validate tokens against BetterAuth JWKS endpoint
- RBAC middleware (`RequireRole`) must check roles from the `orgs` array in JWT claims

## Naming Conventions

### Go
- Packages: lowercase, no underscores (`apidashboard`, `shareddomain`)
- Files: `snake_case.go`
- Types/Interfaces: PascalCase with suffix (`CampaignService`, `OrgRepository`, `AppHandler`)
- Constructors: `NewCampaignService(deps)`
- Errors: `ErrNotFound`, `ErrUnauthorized`, `ErrInvalidInput`
- Tests: `*_test.go`, table-driven, functions `TestCampaignService_Create`

### TypeScript / Next.js
- Components: PascalCase files and exports (`PublisherAppsList.tsx`)
- Hooks: camelCase, `use` prefix (`usePublisherApps.ts`)
- Utils: camelCase (`formatDate.ts`)
- Types: PascalCase, no `I` prefix (`Organization`, `Campaign`)
- Constants: `SCREAMING_SNAKE_CASE`

### Database
- Tables: `snake_case`, plural (`organizations`, `campaigns`, `api_keys`)
- PKs: `id UUID`
- FKs: `{entity}_id` (`org_id`, `campaign_id`)
- Timestamps: `created_at`, `updated_at`
- Booleans: `is_active`, `is_revoked`
- JSONB: `targeting`, `config`, `metadata`

### API
- Endpoints: `kebab-case` (`/api/v1/publisher-apps`)
- Versioned: `/api/v1/...`
- Response envelope: `{"data": ..., "error": {"code": "...", "message": "..."}}`

## Dev Commands

```bash
make infra-up       # Start Postgres, MinIO, Rill, OTel, Jaeger
make infra-down     # Stop docker-compose stack
make seed           # Generate 50k session events → Parquet → MinIO
make rill-ui        # Open Rill (http://localhost:9009)
make jaeger-ui      # Open Jaeger (http://localhost:16686)
make minio-ui       # Open MinIO Console (http://localhost:9001)
```

## Tools & Skills

### RTK (Rust Token Killer)

Token-optimized CLI proxy — saves 60-90% tokens on dev operations. Configured via hooks, works transparently (e.g. `git status` → `rtk git status`). See `~/.claude/RTK.md` for commands.

Use `rtk gain` to check token savings analytics.

### ast-index

Use `/ast-index` skill for fast codebase navigation:
- Find classes/types: "find OrganizationService"
- Find usages: "find usages of ErrNotFound"
- Find implementations: "find implementations of OrganizationRepository"
- Project structure: "project map"
- Module dependencies: "module dependencies"

Prefer `/ast-index` over manual Grep/Glob when searching for symbols, types, or architectural patterns.

### Plugins (project scope)

Managed in `.claude/settings.json`. Enable/disable as needed:

| Plugin               | Status   | Purpose                          |
|----------------------|----------|----------------------------------|
| gopls-lsp            | enabled  | Go LSP — diagnostics, completion |
| typescript-lsp       | disabled | TS LSP — enable for frontend     |
| frontend-design      | disabled | `/frontend-design` — quality UI  |
| playwright           | disabled | Playwright E2E test generation   |

Enable: `claude plugin enable <name>`. Disable unused plugins to save context tokens.

### Playwright CLI

Custom skill in `.claude/skills/playwright-cli/`. References for E2E testing:
test generation, element attributes, video recording, tracing, request mocking, session management.

### JetBrains MCP

Project is developed in JetBrains IDE. Use MCP tools when available:
- `build_project` — verify compilation
- `get_file_problems` — find errors/warnings
- `search_symbol` — find type/function definitions
- `execute_run_configuration` — run/debug

### Recommended Skills After Generation

| Skill              | When                                                       |
|--------------------|------------------------------------------------------------|
| `/simplify`        | After large generation — find duplication, improve quality |
| `/security-review` | After auth/multi-tenancy changes — verify isolation        |
| `/review`          | Before merging PR — catch bugs                             |
| `/frontend-design` | Building dashboard UI — enable plugin first                |

## Profile Selection (STRICT)

Each request is processed within ONE profile. Profile is determined combinationally:

1. **Auto-detect** by keywords/context:
   - Bug, error, crash, not working, breaks, exception, stacktrace, 500, regression → **Bug Fix**
   - How, what, where, why, explain, research, investigate, describe, find → **Research**
   - Docs, document, update docs, sync docs, write docs, README → **Update Docs**
   - Add, create, implement, build, new feature, new endpoint, new entity → **Feature**
2. If auto-detected with high confidence (clear keywords match) — **proceed immediately**, no confirmation needed. Log: `[Profile: <name>]`
3. If ambiguous (multiple profiles match or no clear keywords) — confirm via `AskUserQuestion`
4. User can explicitly specify a profile — always proceed immediately
5. **First action after profile selection**: create workspace `reports/<slug>/` + `_status.md` (profiles reference this step but do NOT duplicate it)

### Available Profiles

| Profile     | When to Use                                                        |
|-------------|--------------------------------------------------------------------|
| Bug Fix     | Bug, regression, crash, unexpected behavior, broken endpoint       |
| Feature     | New endpoint, entity, page, component — writing new code           |
| Research    | Codebase investigation, architecture question, coverage analysis   |
| Update Docs | Sync `docs/` with current code state, fill TODOs, add new sections |

### Available Agents

Agent definitions in `.claude/agents/`. Agents are launched via `Agent` tool with their prompt as context.

**Builders (write code):**

| Agent       | File                            | Model  | Role                                           |
|-------------|---------------------------------|--------|------------------------------------------------|
| go-builder      | `.claude/agents/go-builder.md`      | sonnet | Go services, handlers, repos — strict layering |
| ts-builder      | `.claude/agents/ts-builder.md`      | sonnet | Next.js pages, components, hooks               |
| sql-builder     | `.claude/agents/sql-builder.md`     | sonnet | Migrations, sqlc queries                       |
| go-test-writer  | `.claude/agents/go-test-writer.md`  | sonnet | Find uncovered Go modules, write unit tests    |
| e2e-test-writer | `.claude/agents/e2e-test-writer.md` | sonnet | Playwright smoke tests — generates or reads scenarios |
| refactor-go     | `.claude/agents/refactor-go.md`     | sonnet | Architecture violations, SOLID, layering       |

**Experts (investigate, do NOT write code):**

| Agent             | File                                  | Model  | Role                                            |
|-------------------|---------------------------------------|--------|-------------------------------------------------|
| go-diagnostics    | `.claude/agents/go-diagnostics.md`    | sonnet | Trace handler→service→repo→SQL, find root cause |
| ts-diagnostics    | `.claude/agents/ts-diagnostics.md`    | sonnet | Trace page→component→hook→API, find root cause  |
| sql-analyzer      | `.claude/agents/sql-analyzer.md`      | sonnet | Schema, queries, indexing, multi-tenancy audit  |
| rill-analyzer     | `.claude/agents/rill-analyzer.md`     | sonnet | Rill dashboards, models, sources, metrics       |
| security-reviewer | `.claude/agents/security-reviewer.md` | sonnet | OWASP + multi-tenancy isolation audit           |
| git-investigator  | `.claude/agents/git-investigator.md`  | sonnet | Git history, blame, regression search           |
| docs-analyzer     | `.claude/agents/docs-analyzer.md`     | sonnet | Code vs docs comparison, gap detection          |

**Utility:**

| Agent            | File                                 | Model  | Role                                   |
|------------------|--------------------------------------|--------|----------------------------------------|
| test-runner      | `.claude/agents/test-runner.md`      | sonnet | Run Go/TS/SQL checks, analyze failures |
| report-writer    | `.claude/agents/report-writer.md`    | haiku  | Compile final report in task workspace |
| system-analytics | `.claude/agents/system-analytics.md` | sonnet | Convert feature request into tech spec |

---

## Task Workspace

Each task gets a dedicated directory in `./reports/` for inter-agent communication and cross-session recovery.

### Structure

```
reports/<slug>/
  _status.md              ← current state + handoff, replaced at each transition

  # Bug Fix example:
  01-reproduce.md         ← main
  02-diagnose-go.md       ← go-diagnostics
  02-diagnose-sec.md      ← security-reviewer
  02-diagnose-git.md      ← git-investigator
  03-fix.md               ← go-builder
  04-test-go.md           ← go-test-writer
  04-test-e2e.md          ← e2e-test-writer
  05-validate.md          ← test-runner
  06-report.md            ← report-writer

  # Feature example:
  01-spec.md              ← main / system-analytics
  02-implement-go.md      ← go-builder
  02-implement-sql.md     ← sql-builder
  02-implement-ts.md      ← ts-builder
  03-test-go.md           ← go-test-writer
  03-test-e2e.md          ← e2e-test-writer
  04-validate.md          ← test-runner
  05-report.md            ← report-writer

  # Research example:
  01-explore-go.md        ← go-diagnostics
  01-explore-docs.md      ← docs-analyzer
  02-analyze.md           ← main (synthesis)
  03-report.md            ← report-writer

  # Update Docs example:
  01-analyze-docs.md      ← docs-analyzer
  01-analyze-go.md        ← go-diagnostics
  01-analyze-ts.md        ← ts-diagnostics
  02-plan.md              ← main
  03-update.md            ← main (docs changes made)
  04-validate.md          ← docs-analyzer
  05-report.md            ← report-writer
```

### `_status.md` Format

```markdown
# Task: <title>
Profile: Bug Fix | Feature | Research | Update Docs
Stage: <current stage>
Next: <agent or action>
Created: <YYYY-MM-DD>
Updated: <YYYY-MM-DD HH:MM>

## Context
<brief description of the task>

## Handoff
next: <agent name>
reason: <why this agent is next>
input: <what the agent needs to know — file paths, root cause, etc.>
```

`_status.md` is **replaced entirely** at each stage transition (not appended).

### Rules

1. **Main creates workspace** at task start: `mkdir -p reports/<slug>/` + write `_status.md`
2. **Each agent writes its own file** — numbered by stage, suffixed by agent name
3. **Agents read previous files** for context instead of receiving a rephrased summary from main
4. **Main passes workspace path** to each agent in the prompt: "Workspace: `./reports/<slug>/`. Read previous stage files for context."
5. **Parallel agents** write to separate files (e.g., `02-diagnose-go.md`, `02-diagnose-sec.md`) — no race conditions
6. **Loop iterations**: when revisiting a stage, agent **overwrites** its file (e.g., `03-fix.md` is replaced, not duplicated as `03-fix-v2.md`)
7. **Empty agent result**: if an agent returns with no findings, main notes "no issues found" and proceeds — do not re-launch or block
8. **Cross-session recovery**: new session scans `reports/*/_status.md` for tasks where Stage is not `Done` → offers to continue

### Slug Convention

`<short-description>-<profile>` — e.g., `org-getbyid-bug`, `campaigns-crud-feature`, `auth-flow-research`, `backend-docs`

---

## Profile: Bug Fix

### Workflow (STRICT)

```
Reproduce → Diagnose → Fix → Test → Validate → Report → Done
```

#### Allowed Transitions

```
Reproduce  → Diagnose
Reproduce  → Report           (bug not reproducible — report with mark)
Diagnose   → Fix
Diagnose   → Reproduce        (need to reproduce differently)
Diagnose   → Report           (diagnosis only, fix not required/possible)
Fix        → Test
Fix        → Diagnose         (fix revealed different root cause)
Test       → Validate
Validate   → Report           (all checks pass)
Validate   → Fix              (fix doesn't work — loops Fix → Test → Validate)
Validate   → Test             (tests themselves are wrong)
Validate   → Diagnose         (root cause was different)
Report     → Done
```

All other transitions FORBIDDEN. Before changing stage: `[Stage: X → Y]`.

#### Agent Launch Policy (MANDATORY)

Agent launch at each stage is **MANDATORY, not optional**. Even for trivial bugs — agents may find related issues that main misses. Main MUST NOT perform agent work itself (diagnosing, writing fixes, running checks). Main orchestrates; agents execute.

Violations:
- Main reads code to find root cause instead of launching `go-diagnostics` → **FORBIDDEN**
- Main writes a fix instead of launching `go-builder` → **FORBIDDEN**
- Main writes tests instead of launching `go-test-writer` → **FORBIDDEN**
- Main runs `go build`/`go test` instead of launching `test-runner` → **FORBIDDEN**
- Skipping `git-investigator` because "the bug is obvious" → **FORBIDDEN**

If an agent returns and confirms the bug is trivial, main may note that in the report — but the agent MUST still run.

#### Agents by Stage

| Stage     | Agents (parallel, select by stack)                                            | Role                          |
|-----------|-------------------------------------------------------------------------------|-------------------------------|
| Reproduce | main                                                                          | Run tests, curl, read logs    |
| Diagnose  | (`go-diagnostics` or `ts-diagnostics`) + `git-investigator` + `security-reviewer` | Parallel investigation    |
| Fix       | `go-builder` or `ts-builder` or `sql-builder`                                 | Write fix                     |
| Test      | `go-test-writer` or `e2e-test-writer`                                         | Write regression test for bug |
| Validate  | `test-runner`                                                                 | Run all checks                |
| Report    | `report-writer`                                                               | Save to workspace             |

#### Main's Job at Each Stage

**Reproduce:** Main CAN read code and run tests at this stage (it's the entry point).
1. Get bug description (from user, ticket, log)
2. Determine affected service by analyzing bug description, file paths, stack trace. Only escalate to user if zero clues after reading code
3. Attempt reproduction (run tests, curl endpoints, read logs)
4. Write `01-reproduce.md` to workspace
5. If NOT reproducible after 3 attempts — ask user or move to Report with "Not Reproduced" mark

**Diagnose:** Main does NOT investigate. Main:
1. Launches agents **in parallel**, each with workspace path
2. Waits for results (agents write `02-diagnose-*.md` to workspace)
3. Reads agent files, forms root cause hypothesis
4. If multiple hypotheses — pick most likely, document reasoning in workspace. Only escalate to user if hypotheses are equally probable

**Fix:** Main does NOT write code. Main:
1. Launches appropriate builder agent with workspace path (agent reads diagnose files for context)
2. Agent writes fix + `03-fix.md` to workspace
3. If fix touches auth/multi-tenancy — flag for security review

#### What Agents Do (context for agent prompts)

- `go-diagnostics`: trace handler→service→repo→SQL, find root cause
- `git-investigator`: recent changes in affected area, blame, **who introduced the bug** (commit + author for context)
- `security-reviewer`: auth/multi-tenancy implications
- `go-builder`/`ts-builder`/`sql-builder`: minimal change — fix the bug, do not refactor surrounding code. **Fix the code, not the test** — never weaken tests to make them pass

**Test:** Main does NOT write tests. Main:
1. Launches `go-test-writer` or `e2e-test-writer` with workspace path (agent reads fix files for context)
2. Agent writes regression test covering the bug scenario + `04-test-go.md` or `04-test-e2e.md` to workspace
3. Regression test MUST assert the fixed behavior — if the fix is reverted, the test MUST fail

**Validate:** Main does NOT run checks. Main launches `test-runner` with workspace path. Agent determines checks by stack:

| Stack                          | Checks                                               |
|--------------------------------|------------------------------------------------------|
| Go (`services/`, `packages/`)  | `go build ./...` → `go vet ./...` → `go test ./...`  |
| TypeScript (`apps/dashboard/`) | `pnpm typecheck` → `pnpm lint` → `pnpm test` → `playwright e2e` |
| SQL (`infra/migrations/`)      | `sqlc generate` — verify queries compile             |
| Auth/RBAC changes              | `/security-review`                                   |

Agent writes `05-validate.md` to workspace. If checks fail, main reads error details and decides:
- Build/vet fails → back to **Fix** (source code is broken)
- Test assertion fails → back to **Fix** (fix didn't solve the bug, or broke something else)
- Test compilation error → back to **Test** (test code itself is broken)
- E2E failure (element not found, timeout) → back to **Fix** (UI is broken) or **Test** (selector is wrong)

**Loop until green.** If 3 iterations without progress → escalate to user with summary of what's failing.

**Report:**
`report-writer` compiles `06-report.md` in the workspace from all previous stage files. Updates `_status.md`: `Stage: Done`.

---

## Profile: Feature

### Workflow (STRICT)

```
Spec → Implement → Test → Validate → Report → Done
```

#### Allowed Transitions

```
Spec       → Implement
Implement  → Test
Implement  → Spec             (spec was incomplete, need clarification)
Test       → Validate
Test       → Implement        (tests reveal missing implementation)
Validate   → Report           (all checks pass)
Validate   → Implement        (build/lint fails)
Validate   → Test             (tests fail)
Report     → Done
```

All other transitions FORBIDDEN. Before changing stage: `[Stage: X → Y]`.

#### Agent Launch Policy (MANDATORY)

Agent launch at each stage is **MANDATORY, not optional**. Main orchestrates; agents execute.

Violations:
- Main writes Go/TS/SQL code instead of launching builder agents → **FORBIDDEN**
- Main writes tests instead of launching test-writer agents → **FORBIDDEN**
- Main runs `go build`/`go test` instead of launching `test-runner` → **FORBIDDEN**
- Skipping Test stage without justification → **FORBIDDEN**
- Exception: SQL-only changes (migrations, indexes) with no new Go/TS code — Test stage may be skipped with `[Skip Test: SQL-only, no testable code]` log

#### Agents by Stage

| Stage     | Agents (parallel by stack)                                                     | Role                              |
|-----------|--------------------------------------------------------------------------------|-----------------------------------|
| Spec      | main or `system-analytics`                                                     | Write technical spec              |
| Implement | `go-builder` + `sql-builder` + `ts-builder` (parallel, by stack)               | Write code by layer               |
| Test      | `go-test-writer` + `e2e-test-writer` (parallel, by stack)                      | Write tests for new code          |
| Validate  | `test-runner`                                                                  | Run all checks                    |
| Report    | `report-writer`                                                                | Save to workspace                 |

#### Main's Job at Each Stage

**Spec:** Main writes technical spec or launches `system-analytics` for complex features.
1. Understand user request — what entity/endpoint/page to build
2. Determine affected stacks (Go, SQL, TypeScript)
3. Follow New Entity Checklist order if adding a new entity
4. Write `01-spec.md` to workspace with: what to build, which layers, acceptance criteria

**Implement:** Main does NOT write code. Main:
1. Determines which stacks are affected from spec
2. Launches builder agents by stack, each with workspace path
3. If SQL + Go/TS: launch `sql-builder` first, **wait for it to finish**, then launch `go-builder` and/or `ts-builder` in parallel
4. If single stack (e.g., TS-only): launch that builder directly
5. Agents read spec from workspace for context
6. Agents write `02-implement-*.md` to workspace

**Test:** Main does NOT write tests. Main:
1. Launches `go-test-writer` and/or `e2e-test-writer` with workspace path
2. Agents read implement files to understand what was built
3. Agents write tests + `03-test-*.md` to workspace
4. `go-test-writer`: table-driven unit tests for services, handlers, middleware
5. `e2e-test-writer`: smoke scenarios for new UI features

**Validate:** Main launches `test-runner`. Agent writes `04-validate.md`. If checks fail, main reads error details and decides:
- Build/vet/lint fails → back to **Implement** (source code is broken)
- Test assertion fails → back to **Implement** (implementation doesn't match expected behavior)
- Test compilation error → back to **Test** (test code itself is broken)
- E2E failure (element not found, timeout) → back to **Implement** (UI is broken) or **Test** (selector is wrong)

**Loop until green.** If 3 iterations without progress → escalate to user.

**Report:**
`report-writer` compiles `05-report.md` from all stage files. Updates `_status.md`: `Stage: Done`.

---

## Profile: Research

### Workflow (STRICT)

```
Explore → Analyze → Report → Done
```

#### Allowed Transitions

```
Explore   → Analyze
Explore   → Report            (ONLY after agents ran and confirmed trivial)
Analyze   → Explore           (need to investigate deeper)
Analyze   → Report
Report    → Done
```

All other transitions FORBIDDEN. Before changing stage: `[Stage: X → Y]`.

#### Agent Launch Policy (MANDATORY)

Agent launch at Explore stage is **MANDATORY, not optional**. Even for simple questions — agents may find related gaps, inconsistencies, or undocumented behavior that main would miss. Main orchestrates and synthesizes; agents investigate.

Violations:
- Main reads code and traces data flow instead of launching topic-relevant agents → **FORBIDDEN**
- Main skips `docs-analyzer` because "there are no docs for this" → **FORBIDDEN**
- Main writes the report without launching `report-writer` → **FORBIDDEN**

Select agents by topic relevance (not all 5 every time), but at least 2 agents MUST run in parallel at Explore stage.

#### Agents by Stage

| Stage   | Agents (parallel, by topic)                                                              | Role                                     |
|---------|------------------------------------------------------------------------------------------|------------------------------------------|
| Explore | `go-diagnostics` + `ts-diagnostics` + `sql-analyzer` + `rill-analyzer` + `docs-analyzer` | Parallel investigation (select by topic) |
| Analyze | main                                                                                     | Synthesize agent findings                |
| Report  | `report-writer`                                                                          | Save to workspace                        |

#### Main's Job at Each Stage

**Explore:** Main does NOT read code or trace flows. Main:
1. Selects relevant agents by topic (at least 2)
2. Launches agents in parallel, each with: workspace path + question + scope
3. Waits for results

**Analyze:** Main reads agent output files from workspace, synthesizes into structured answer. Writes `02-analyze.md` to workspace.

**Report:** Main launches `report-writer` with workspace path.

#### What Agents Do (context for agent prompts)

Agents at Explore stage independently:
- Use `/ast-index` for symbol search, project structure, module dependencies
- Read relevant files (handlers, services, repositories, migrations, configs)
- Trace data flow: HTTP request → handler → service → repository → SQL
- Check `docs/` for existing documentation on the topic
- Write findings to their workspace file (e.g., `01-explore-go.md`)

**CRITICAL: Research profile MUST NOT modify any code files.** Read-only investigation.

**Analyze:** Main synthesizes agent findings and classifies each finding by severity:
- **CRITICAL** — security gap, data leak, broken invariant
- **IMPORTANT** — missing tests, undocumented behavior, inconsistency
- **NOTE** — minor observation, potential improvement

Include a **"Not investigated (consciously)"** section — explicitly list areas that were out of scope and why.

**Report:**
`report-writer` compiles `03-report.md` in the workspace from all stage files. Updates `_status.md`: `Stage: Done`.

---

## Profile: Update Docs

### Workflow (STRICT)

```
Analyze Code → Plan Changes → Update Docs → Validate → Report → Done
```

#### Allowed Transitions

```
Analyze Code   → Plan Changes
Plan Changes   → Update Docs
Update Docs    → Validate
Update Docs    → Analyze Code    (discovered undocumented area)
Validate       → Report
Validate       → Update Docs    (broken links, missing sections)
Report         → Done
```

All other transitions FORBIDDEN. Before changing stage: `[Stage: X → Y]`.

#### Agent Launch Policy (MANDATORY)

Agent launch at each stage is **MANDATORY, not optional**. Main orchestrates and writes docs; agents investigate and validate.

Violations:
- Main reads code to compare with docs instead of launching `docs-analyzer` + `go-diagnostics` / `ts-diagnostics` → **FORBIDDEN**
- Main skips Validate stage or checks links manually instead of launching `docs-analyzer` → **FORBIDDEN**
- Main writes the report without launching `report-writer` → **FORBIDDEN**

All agents listed in the "Agents by Stage" table for a given stage MUST be launched. No shortcuts.

#### Agents by Stage

| Stage        | Agents                                                           | Role                   |
|--------------|------------------------------------------------------------------|------------------------|
| Analyze Code | `docs-analyzer` + `go-diagnostics` + `ts-diagnostics` (parallel) | Compare code vs docs   |
| Plan Changes | main                                                             | List changes           |
| Update Docs  | main                                                             | Write docs             |
| Validate     | `docs-analyzer`                                                  | Check links, structure |
| Report       | `report-writer`                                                  | Save to workspace      |

**CRITICAL: Update Docs profile MUST NOT modify source code.** Only `docs/` files.

#### Main's Job at Each Stage

**Analyze Code:** Main does NOT read code or compare with docs. Main:
1. Launches agents in parallel, each with workspace path
2. Waits for results (agents write `01-analyze-*.md` to workspace)
3. Reads agent files

**Plan Changes:** Main reads agent analysis files from workspace, then:
1. Lists docs sections to create or update
2. Log plan: `[Plan: N sections to update: ...]` — proceed immediately
3. Prioritize: fill TODOs > update stale content > add new sections
4. Write `02-plan.md` to workspace

**Update Docs:** Main writes docs following:
- Existing docs style (Russian content, English headers, table navigation)
- `docs/backend/` ↔ `services/api-dashboard/`, `services/api-sdk/`
- `docs/dashboard/` ↔ `apps/dashboard/`
- `docs/sdk/` ↔ SDK public API and protocol
- Update `docs/glossary.md` if new domain terms introduced

**Validate:** Main does NOT check links manually. Main launches `docs-analyzer` with workspace path. Agent checks:
1. All relative links in changed docs point to existing files
2. New pages are referenced from parent README.md
3. `docs/README.md` navigation table is up to date
4. No orphan pages (every .md is reachable from README)

#### What Agents Do (context for agent prompts)

- `docs-analyzer`: compare docs/ with current code, find gaps, stale sections, TODOs
- `go-diagnostics`: read Go services, extract public API surface for documentation
- `ts-diagnostics`: read frontend code, extract component/hook structure for documentation

**Report:**
`report-writer` compiles final report in the workspace from all previous stage files. Updates `_status.md`: `Stage: Done`.

---

## Detailed Rules

Go patterns, anti-patterns, and code examples are in `.claude/rules/` — loaded automatically by file glob.
