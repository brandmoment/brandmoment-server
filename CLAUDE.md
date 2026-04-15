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

| Tool | Check command | Required for |
|------|---------------|-------------|
| Go 1.23+ | `go version` | Backend services |
| sqlc | `sqlc version` | DB query codegen |
| golang-migrate | `migrate -version` | DB migrations |
| Docker | `docker --version` | Infra (Postgres, MinIO, Rill, OTel) |
| pnpm 9.15+ | `pnpm --version` | Frontend, monorepo |
| Node 20+ | `node --version` | Frontend |

If any tool is missing — **stop and report** which tools need to be installed. Suggest install commands (e.g. `brew install go`). Do NOT proceed with code generation without required tools.

## Post-generation Checks

After generating code, always verify:

1. `go build ./...` — compiles without errors
2. `go vet ./...` — no static analysis issues
3. `go test ./...` — all tests pass
4. Run `/simplify` — check for code duplication and quality
5. Run `/security-review` — check multi-tenancy isolation and auth

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

## Detailed Rules

Go patterns, anti-patterns, and code examples are in `.claude/rules/` — loaded automatically by file glob.
