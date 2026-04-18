# Docs Analysis: backend/ + dashboard/ + platform/ vs Code + Spec

Agent: docs-analyzer
Date: 2026-04-18

---

## 1. Scope of Analysis

Files read:
- `docs/backend/README.md` — skeleton with 3 TODO sections
- `docs/dashboard/README.md` — skeleton with 3 TODO sections
- `docs/platform/README.md` — skeleton with 4 TODO sections (Overview is also TODO)
- `server-dashboard-spec.md` — full v1 spec (17 sections)
- `infra-architecture.md` — infrastructure layer-by-layer analysis (12 layers)
- `docs/architecture.md` — high-level arch, partially filled
- `docs/glossary.md` — domain terms, SDK-focused
- `services/api-dashboard/` — fully read (all layers: config, router, middleware, handler, service, repository, model, httputil)
- `packages/shared-domain/` — sqlc schema, queries, generated db code
- `infra/migrations/` — only 1 migration exists (organizations)
- `apps/dashboard/` — empty directory, nothing implemented

---

## 2. Stale Content

### docs/backend/README.md — Overview section is stale

Current overview describes a future AI matching system that does not exist in code:
- "AI Matching Engine — подбор highest-fit спонсора за миллисекунды" — not implemented; spec §2 explicitly marks AI matching as out of scope for v1
- "Continuous learning от каждой сессии" — not implemented
- "Transparent Decisions — каждое решение AI логируется с объяснением (explainability)" — not implemented; matching_decision table does not exist yet (no migration)
- "Publisher Rules Engine" — no code exists (no publisher_rule table, no handler, no service)
- Link to `../sdk/api-spec.md` as "Sponsor Selection API" is accurate as a link but misleading in context — it refers to a future SDK API, not the current dashboard API

The overview section was written from product vision, not from current implementation state. A developer reading this to understand `services/api-dashboard/` will be confused about what is actually built.

### docs/architecture.md — Component table is stale

The "Backend API" row describes "AI Matching Engine — контекстный подбор спонсора" which is a future capability. Current implementation is CRUD + JWT auth for organizations only. The architecture diagram shows only Mobile SDK → Backend API → Data Pipeline flow, missing the actual current primary flow: Dashboard → api-dashboard → Postgres.

### docs/glossary.md — Terms reflect SDK product, not current server implementation

All existing terms are SDK/product-layer terms. Missing server-implementation terms that a developer needs (see section 5 below).

---

## 3. TODO Placeholders (all unfilled)

| File | Section | Status |
|------|---------|--------|
| `docs/backend/README.md` | Architecture | `<!-- TODO -->` — empty |
| `docs/backend/README.md` | Guidelines | `<!-- TODO -->` — empty |
| `docs/backend/README.md` | Roadmap | `<!-- TODO -->` — empty |
| `docs/dashboard/README.md` | Architecture | `<!-- TODO -->` — empty |
| `docs/dashboard/README.md` | Guidelines | `<!-- TODO -->` — empty |
| `docs/dashboard/README.md` | Roadmap | `<!-- TODO -->` — empty |
| `docs/platform/README.md` | Overview | `<!-- TODO: cloud provider, environments, deployment flow -->` |
| `docs/platform/README.md` | Architecture | `<!-- TODO -->` — empty |
| `docs/platform/README.md` | Guidelines | `<!-- TODO -->` — empty |
| `docs/platform/README.md` | Roadmap | `<!-- TODO -->` — empty |
| `docs/architecture.md` | Environments | `<!-- TODO: dev / staging / prod -->` |

Total: 11 unfilled TODO sections across 4 files.

---

## 4. Missing Documentation (code exists, docs don't)

### 4.1 docs/backend/ — Architecture section (highest priority)

What must be documented based on actual code in `services/api-dashboard/`:

**Service structure and layering:**
- Directory layout: `cmd/server/main.go`, `internal/{config,router,handler,service,repository,model,httputil,middleware}/`
- Layering rule: handler decodes/encodes only, service holds business logic + OTel + slog, repository wraps sqlc-generated Queries only, model holds domain types and sentinel errors
- Dependency injection: constructors wired in `main.go` (no global state, no init())
- `httputil` package: shared `RespondJSON` / `RespondError` used by all handlers and middleware — not duplicated

**JWT auth flow (actual implementation, not spec):**
- `middleware.Auth` constructed from `JWT_SECRET` env var (symmetric HMAC, not JWKS yet — important gap vs spec)
- `ValidateJWT`: extracts `Authorization: Bearer <token>`, parses with `golang-jwt/jwt/v5`, validates `X-Org-ID` header against `orgs[]` claim, stores `org_id` + `role` + all `org_ids` in context
- `RequireRole`: reads role from context, compares against allowed list
- Context helpers: `OrgIDFromContext`, `RoleFromContext`, `OrgIDsFromContext`
- Current implementation uses symmetric HMAC secret, not BetterAuth JWKS — this is a known gap vs spec §11 and needs to be documented as a TODO

**Organizations CRUD (only implemented entity):**
- `POST /api/v1/organizations` — create org (type: admin|publisher|brand, name, slug)
- `GET /api/v1/organizations` — list orgs the calling user is a member of (filtered by JWT org_ids)
- `GET /api/v1/organizations/{id}` — get single org (access-checked against JWT org_ids)
- All endpoints behind `ValidateJWT` + `RequireRole("viewer","editor","admin","owner")`
- No `PUT /DELETE` implemented yet — no update or delete organization

**OpenTelemetry wiring:**
- OTLP gRPC exporter, endpoint from `OTEL_EXPORTER_OTLP_ENDPOINT` env (default: `localhost:4317`)
- `otelchi.Middleware("api-dashboard")` on router — all HTTP requests traced
- Each service method starts its own span: `s.tracer.Start(ctx, "OrganizationService.Create")`
- `span.RecordError(err)` on failures

**Configuration (env vars):**
- `PORT` (default: 8080)
- `DATABASE_URL` (required)
- `JWT_SECRET` (required)
- `OTEL_EXPORTER_OTLP_ENDPOINT` (default: localhost:4317)

**Response envelope:**
- `{"data": <any>}` on success
- `{"error": {"code": "SCREAMING_SNAKE_CASE", "message": "<human string>"}}` on error
- HTTP status codes: 200 (GET), 201 (POST), 400 (bad input), 401 (no/invalid JWT), 403 (wrong org or role), 404 (not found), 500 (internal)

### 4.2 docs/backend/ — Guidelines section

What must be documented based on actual patterns in code:

**Testing pattern:**
- Table-driven tests in `internal/service/*_test.go`
- Mock interfaces defined inline in test file (no testify, no gomock — pure Go)
- `noop.NewTracerProvider()` from `go.opentelemetry.io/otel/trace/noop` for OTel in tests
- Pattern: `mockOrgRepo` struct implementing the repository interface with func fields (`insertFn`, `getByIDFn`, etc.)

**Error handling pattern:**
- Sentinel errors in `model/` package: `ErrNotFound`, `ErrUnauthorized`, `ErrInvalidInput`
- Repository wraps pgx.ErrNoRows → `model.ErrNotFound`
- Service wraps errors with `fmt.Errorf("insert organization: %w", err)`
- Handler uses `handleServiceError()` to map model errors to HTTP status codes

**sqlc usage:**
- All queries in `packages/shared-domain/queries/<entity>.sql`
- Generated code in `packages/shared-domain/db/`
- Repository never writes raw SQL — only calls `r.q.GeneratedMethod(ctx, params)`
- pgtype.UUID conversions: `uuidToPgtype()` and `pgtypeToUUID()` helpers in repository

**Adding a new entity (step-by-step):**
1. Migration in `infra/migrations/` (up + down)
2. sqlc query in `packages/shared-domain/queries/<entity>.sql`, run `sqlc generate`
3. Model struct + sentinel errors in `internal/model/<entity>.go`
4. Repository interface + implementation in `internal/repository/<entity>.go`
5. Service with OTel tracing + slog in `internal/service/<entity>.go`
6. Handler in `internal/handler/<entity>.go`, uses `handleServiceError` and `httputil.RespondJSON/RespondError`
7. Tests in `internal/service/<entity>_test.go`
8. Register handler in `internal/router/router.go` under `/api/v1`

### 4.3 docs/backend/ — Roadmap section

What v1 roadmap should contain based on spec §2, §7, §9, §17:

**Implemented (done):**
- Monorepo skeleton (Turborepo, go.work, docker-compose)
- Rill + seed data (50k events in Parquet)
- organizations table migration + CRUD (GET list, GET by ID, POST create)
- JWT auth middleware (symmetric HMAC)
- OTel tracing wired (OTLP gRPC)

**Not implemented yet (v1 scope from spec):**
- `packages/proto/dashboard.yaml` — OpenAPI 3.1 spec (source of truth not created yet)
- Go server stub codegen (oapi-codegen) — blocked by proto
- TypeScript client codegen (openapi-typescript) — blocked by proto
- Identity tables: `user`, `org_membership` (no migration)
- Publisher domain: `publisher_app`, `api_key`, `publisher_rule` (no migrations, no CRUD)
- Brand domain: `campaign`, `creative` (no migrations, no CRUD)
- Matching audit: `matching_decision` (no migration)
- BetterAuth JWKS validation (currently symmetric HMAC — not production-ready)
- `/v1/me` endpoint
- `/v1/orgs/:id/invites` endpoint
- API key CRUD (create, revoke)
- Publisher rules CRUD
- Campaign CRUD + status patch
- Creative upload (multipart → S3 → HTML5 validation)
- Rill embed signed URL endpoint
- DuckDB aggregations from Parquet → REST for charts
- Next.js dashboard (completely empty)

**Out of scope v1 (per spec §2, §16):**
- SDK hot-path API (`/v1/sdk/init`, `/v1/session/*`) — separate future service
- AI matching model — stub/rules-based placeholder planned, real ML in future
- Real-time event ingestion pipeline (Kafka/Redpanda)
- Billing (Stripe)
- Static/video creatives
- Enterprise SSO

### 4.4 docs/dashboard/ — Architecture section

What must be documented:

**Current state:**
- `apps/dashboard/` directory exists but is completely empty — no Next.js scaffold yet
- BetterAuth has not been set up
- No pages, no components, no hooks

**Planned architecture (from spec §4, §10):**
- Next.js 15 App Router + React 19 + TypeScript
- BetterAuth self-hosted within Next.js (auth endpoints, session management, JWT issuance for Go API)
- shadcn/ui + Tailwind v4 as design system
- TanStack Query for data fetching, TanStack Table for tabular data
- Recharts for custom analytics charts (user-facing, not Rill)
- Rill embedded via iframe served through Next.js reverse proxy with signed URLs (org-scoped)
- Org-type-aware sidebar: publisher sees apps/rules/analytics; brand sees campaigns/creatives/analytics; admin sees all orgs

**Page structure planned (spec §10):**
- `/login`, `/signup`, `/accept-invite/:token`
- `/onboarding`
- `/apps` (publisher)
- `/campaigns` (brand)
- `/analytics` (both roles, Rill iframe)
- `/admin/*` (admin org)

### 4.5 docs/dashboard/ — Guidelines section

What must be documented:

- Component file naming: PascalCase (`PublisherAppsList.tsx`)
- Hook naming: camelCase with `use` prefix (`usePublisherApps.ts`)
- Type naming: PascalCase, no `I` prefix
- API client: openapi-typescript generated client + openapi-fetch wrapper
- Auth in Next.js: BetterAuth client SDK, session accessed via `auth()` in Server Components
- JWT forwarding: Next.js server actions / route handlers attach `Authorization: Bearer <token>` + `X-Org-ID` header to Go API calls
- Rill iframe security: never expose Rill URL directly; always proxy through `/api/rill/embed` with session check + signed URL generation

### 4.6 docs/dashboard/ — Roadmap section

**Not started:**
- Next.js 15 scaffold with BetterAuth
- Auth pages (login, signup, invite accept)
- Org switcher + role-aware layout
- Publisher pages (apps, API keys, rules)
- Brand pages (campaigns, creatives upload + preview)
- Analytics pages (Rill iframe + Recharts custom charts)
- Admin pages

### 4.7 docs/platform/ — All sections (entirely empty)

What must be documented based on `infra/` directory and `infra-architecture.md`:

**Overview (local dev):**
- Docker Compose stack: Postgres 17, MinIO (S3-compatible), Rill Developer, OTel Collector, Jaeger
- Make commands: `make infra-up`, `make infra-down`, `make seed`, `make rill-ui`, `make jaeger-ui`, `make minio-ui`
- Default ports: Postgres 5432, MinIO API 9000, MinIO Console 9001, Rill 9009, Jaeger UI 16686, OTel gRPC 4317
- OTel Collector config: `infra/docker/otel-collector-config.yaml`

**Architecture (target, from infra-architecture.md):**
- 12-layer target diagram (Edge → Ingress → App → Data → Stream → BI → Obs)
- Decisions already made: Go + Next.js 15 + BetterAuth + Rill self-hosted + OTel + Turborepo
- Open decisions: CDN/Edge (Cloudflare+R2 recommended), hosting pattern (Option A: Vercel+Fly.io+Neon recommended), Postgres managed (Neon recommended), Redis (Upstash for MVP), observability umbrella (Grafana Cloud recommended)
- Three hosting options with cost estimates: Option A ($0-50/mo), Option B (AWS, $300+/mo), Option C (Hetzner k8s, $50-150/mo)

**Guidelines (local dev):**
- Prerequisites: Go 1.23+, sqlc, golang-migrate, Docker, pnpm 9.15+, Node 20+
- Running migrations: `migrate -path infra/migrations -database $DATABASE_URL up`
- Running sqlc codegen: `sqlc generate` from `packages/shared-domain/`
- Environment variables required for api-dashboard: `DATABASE_URL`, `JWT_SECRET`, `OTEL_EXPORTER_OTLP_ENDPOINT`

**Roadmap (open decisions from infra-architecture.md §Open Questions):**
- Edge/CDN selection
- Hosting pattern selection (Option A/B/C)
- Managed Postgres selection
- Redis/cache selection
- Streaming pipeline v2 (Redpanda vs NATS)
- OLAP v2 (ClickHouse Cloud vs self-host)
- Observability backend (Grafana Cloud vs SigNoz)
- Secrets management (Doppler / Infisical / 1Password CLI)

---

## 5. Missing Glossary Terms

The following terms appear in code or spec but are absent from `docs/glossary.md`:

| Term | Where Used | Definition to Add |
|------|-----------|-------------------|
| Organization | `model.Organization`, migration, spec §7.1 | Мультитенантная единица платформы. Тип: `admin`, `publisher` или `brand`. Каждый пользователь принадлежит одной или нескольким org через `org_membership`. |
| Org Type | spec §7.1, middleware JWT claims | Тип организации: `admin` (внутренняя команда), `publisher` (разработчик приложения), `brand` (рекламодатель). Определяет доступные разделы дашборда и права доступа. |
| Publisher App | spec §7.2, planned entity | Приложение (bundle), зарегистрированное паблишером. Идентифицируется по `bundle_id` и `platform` (ios/android/unity). Привязано к org паблишера. |
| API Key | spec §7.2 | Ключ для аутентификации SDK на горячем пути. Хранится как argon2-хеш; при создании возвращается один раз в открытом виде. Принадлежит `publisher_app`. |
| Publisher Rule | spec §7.2, planned entity | Ограничения паблишера на матчинг: блоклисты категорий и брендов, frequency cap, geo и platform allowlist. Применяются при подборе спонсора. |
| Campaign | spec §7.3, planned entity | Рекламная кампания бренда: targeting (категории, гео, платформы), бюджет, статус (draft/active/paused/completed). Содержит HTML5 creatives. |
| HTML5 Creative | spec §7.3 | Набор ассетов кампании в формате HTML5 bundle. Загружается через multipart upload, валидируется в sandbox iframe, хранится в S3/R2. |
| Matching Decision | spec §7.4 | Запись о решении AI-матчинга: какой кампании была назначена сессия (или fallback), с объяснением факторов. Write-only со стороны SDK API. |
| X-Org-ID | middleware/auth.go | HTTP-заголовок запроса: UUID организации, от имени которой действует пользователь. Валидируется против списка org в JWT. |
| OTLP | main.go, docker-compose | OpenTelemetry Protocol. Протокол экспорта трейсов/метрик/логов. В dev трейсы идут в OTel Collector → Jaeger по gRPC на порт 4317. |
| sqlc | shared-domain/ | Инструмент кодогенерации Go-кода из SQL-запросов. Источник правды — `.sql`-файлы в `packages/shared-domain/queries/`. |
| Seed Data | infra/seed/ | Синтетические session events (50k записей), сгенерированные скриптом и загруженные в MinIO/S3 в формате Parquet. Используются Rill для построения дашбордов в v1. |

---

## 6. Broken and Suspect Links

| Location | Link | Status |
|----------|------|--------|
| `docs/backend/README.md` line 16 | `../sdk/api-spec.md` | File exists at `docs/sdk/api-spec.md` — link resolves correctly |
| `docs/architecture.md` line 47 | `sdk/api-spec.md` | File exists — resolves correctly |
| `docs/architecture.md` line 67 | `sdk/README.md` | File exists — resolves correctly |
| `docs/README.md` (Tech table) | All 10 section links | All target directories exist — no broken links |

No broken links found. All relative links resolve to existing files.

---

## 7. Code vs Spec Divergences (not documented anywhere)

These are gaps between `server-dashboard-spec.md` and actual implementation that should be captured in docs:

1. **JWT validation method**: Spec §4 and §11 describe BetterAuth JWKS endpoint for JWT validation. Actual code uses symmetric HMAC secret (`JWT_SECRET` env var). This is a deliberate MVP shortcut but is undocumented. A developer inheriting this code would not know validation needs to migrate to JWKS.

2. **API routes prefix**: Spec §9 uses `/v1/...` (no `/api` prefix). Actual router uses `/api/v1/...`. This divergence matters for the TypeScript client generation and any SDK integration.

3. **Organizations access pattern**: Spec §7.1 says access to organizations is controlled by JWT membership (user sees only orgs from their `orgs[]` claim). Actual implementation: `GET /api/v1/organizations` uses `OrgIDsFromContext` which are the parsed JWT org_ids — correct. `GET /api/v1/organizations/{id}` checks `slices.Contains(orgIDs, id)` in handler before calling service — access check is in handler, not service. This is a pattern worth documenting.

4. **No OpenAPI spec exists**: Spec §5 says `packages/proto/dashboard.yaml` is the source of truth. This file does not exist yet. The spec says it's next step #3, but there is no documentation tracking this gap.

5. **`infra/seed/` exists**: The Makefile references `make seed` and CLAUDE.md confirms seed data generator is done, but no documentation in `docs/data/` or `docs/platform/` describes what the seed script generates or how to run it.

---

## 8. Priority Ranking for Documentation Work

Priority is ranked by impact on a developer trying to work on the codebase or create tasks.

**Priority 1 — Blocks all new development:**
- `docs/backend/README.md` → Architecture: document actual service structure, layering, implemented endpoints, JWT flow, env vars, response envelope. Without this, a developer cannot understand what is built or how to add to it.
- `docs/backend/README.md` → Guidelines: document error handling pattern, testing pattern, entity checklist, sqlc usage. Without this, new entities will be implemented inconsistently.

**Priority 2 — Blocks task planning:**
- `docs/backend/README.md` → Roadmap: list what is done vs what remains for v1. Without this, it is impossible to create accurate tickets or understand progress.
- `docs/platform/README.md` → Overview + Guidelines: document local dev setup, docker-compose stack, Make commands, env vars, migration commands. Without this, a new developer cannot run the project.
- Glossary: add the 11 missing terms (Organization, Org Type, Publisher App, API Key, Publisher Rule, Campaign, HTML5 Creative, Matching Decision, X-Org-ID, OTLP, sqlc, Seed Data).

**Priority 3 — Needed before dashboard development starts:**
- `docs/dashboard/README.md` → Architecture: document planned Next.js structure, BetterAuth integration pattern, Rill iframe proxy pattern, page structure. Needed before any frontend work begins.
- `docs/dashboard/README.md` → Guidelines: TypeScript conventions, auth pattern in Next.js, API client usage.
- `docs/dashboard/README.md` → Roadmap: list all unimplemented dashboard pages and features.

**Priority 4 — Infrastructure decisions:**
- `docs/platform/README.md` → Architecture: document target infra diagram (from `infra-architecture.md`) and open decisions. Important for cost planning but does not block development.
- `docs/platform/README.md` → Roadmap: list open infrastructure decisions from `infra-architecture.md`.
- `docs/architecture.md` → Environments: fill the TODO with dev/staging/prod environment description.
- Fix stale content: `docs/backend/README.md` Overview and `docs/architecture.md` Components table to reflect current implementation (CRUD only, no AI matching).
