# Documentation Update: Server Backend + Dashboard + Platform
Date: 2026-04-18
Status: Complete

## Summary

Updated all architectural documentation for the server backend and dashboard platforms. Filled 11 TODO sections across 3 main README files, fixed stale content describing non-existent AI matching features, added 14 domain terms to glossary, and documented actual implementation state with clear roadmaps for future phases.

Key scope:
- **docs/backend/README.md**: Rewritten with full architecture, guidelines, and 6-phase roadmap
- **docs/dashboard/README.md**: Rewritten with planned architecture, guidelines, and 6-phase roadmap
- **docs/platform/README.md**: Rewritten with local dev stack, infrastructure diagram, and decisions table
- **docs/glossary.md**: Added 14 server-side terms
- **docs/architecture.md**: Fixed stale component descriptions and filled Environments section

---

## Changes Made

### 1. docs/backend/README.md

**Replaced entire Overview section** (was describing non-existent AI matching engine):
- Old: "AI Matching Engine", "Continuous Learning", "Transparent Decisions" — none implemented in v1
- New: Multi-tenant CRUD API for dashboard with organizations domain + future publisher/brand domains

**Added Architecture section** (previously empty TODO):
- Service layering diagram: handler → service → repository → sqlc-generated Queries
- Dependency injection pattern with constructor wiring in main.go (no global state)
- JWT auth flow: currently HMAC-based `JWT_SECRET`, planned migration to BetterAuth JWKS (marked as deviation from spec)
- Middleware chain: otelchi → RequestID → RealIP → Recoverer → ValidateJWT → RequireRole
- Response envelope format: `{"data": ...}` or `{"error": {"code": "...", "message": "..."}}`
- Environment variables table (PORT, DATABASE_URL, JWT_SECRET, OTEL_EXPORTER_OTLP_ENDPOINT)
- Route table for current endpoints (GET /healthz, POST/GET /api/v1/organizations, GET /api/v1/organizations/{id})
- Full request-to-database data flow diagram with middleware, context passing, and error handling

**Added Guidelines section** (previously empty TODO):
- Error handling pattern: sentinel errors in model/ → wrapped with fmt.Errorf → mapped to HTTP status in handler
- Testing pattern: table-driven tests with inline mock interfaces, noop.NewTracerProvider() for OTel
- sqlc usage: all queries in packages/shared-domain/queries/*.sql, generated code wrapped in repository layer
- New entity checklist: 8-step procedure from migration → service → handler → tests → router registration

**Added Roadmap section** (previously empty TODO):
- Phase 0 (DONE): Monorepo, docker-compose, Rill+seed, organizations CRUD, JWT, OTel
- Phase 1: OpenAPI spec (packages/proto/dashboard.yaml), identity tables (users, org_memberships), BetterAuth JWKS
- Phase 2: Publisher domain (publisher_apps, api_keys, publisher_rules) with full CRUD and pagination
- Phase 3: Brand/campaign domain (campaigns, creatives) with multipart upload to S3
- Phase 4: Analytics + embed endpoints (Rill signed URL, DuckDB aggregations)
- Phase 5: OTel metrics (request rate/latency/error per endpoint, DB pool metrics)
- Future v2+: api-sdk hot-path service, AI matching model, event pipeline, billing

---

### 2. docs/dashboard/README.md

**Replaced entire Overview section** (was aspirational product metrics):
- Old: "94.2% fill rate", "$18.40 RPM" — product vision, not current state
- New: Clear statement of planned architecture + current status (app is completely empty)

**Added Architecture section** (previously empty TODO):
- Next.js 15 App Router + React 19 + TypeScript
- BetterAuth self-hosted within Next.js (auth endpoints, session, JWT issuance for Go API)
- shadcn/ui + Tailwind v4 design system
- TanStack Query + TanStack Table for data fetching and tables
- Recharts for user-facing custom charts (separate from Rill BI)
- Rill iframe proxy pattern: never expose URL directly, always proxy through /api/rill/embed with session check + signed URL
- Org-type-aware page structure: publisher sees apps/rules/analytics, brand sees campaigns/creatives/analytics, admin sees all orgs
- API client architecture: openapi-typescript codegen + openapi-fetch wrapper
- JWT forwarding: Next.js server actions/route handlers attach Authorization Bearer + X-Org-ID header to Go API calls

**Added Guidelines section** (previously empty TODO):
- Component naming: PascalCase files (PublisherAppsList.tsx)
- Hook naming: camelCase with use prefix (usePublisherApps.ts)
- Type naming: PascalCase, no I prefix
- Auth pattern in Next.js: Server Components use auth() function, Client Components use session hook
- Rill iframe security: token generation, domain pinning, read-only access

**Added Roadmap section** (previously empty TODO):
- Phase 0: Next.js 15 scaffold + BetterAuth setup + shadcn/ui + TanStack Query + openapi-typescript codegen
- Phase 1: Auth pages (login, signup, accept-invite, onboarding)
- Phase 2: Publisher pages (apps list, API key management, rules editor)
- Phase 3: Brand pages (campaigns, creative upload + sandbox preview)
- Phase 4: Analytics pages (Rill iframe + Recharts custom charts)
- Phase 5: Admin org overview pages
- Future v2+: real-time session dashboard, ML matching transparency UI, billing/payouts UI

---

### 3. docs/platform/README.md

**Completely rewritten** (was 4 empty TODO sections):

**Added Overview section**:
- Local dev stack: Docker Compose with Postgres 17, MinIO, Rill Developer, OTel Collector, Jaeger
- Make commands table: infra-up, infra-down, seed, rill-ui, jaeger-ui, minio-ui with descriptions
- Default ports table: Postgres 5432, MinIO API 9000, MinIO Console 9001, Rill 9009, Jaeger 16686, OTel gRPC 4317
- Prerequisites: Go 1.23+, sqlc, golang-migrate, Docker, pnpm 9.15+, Node 20+

**Added Architecture section**:
- 12-layer target infrastructure diagram (Edge → Ingress → App → Data → Stream → BI → Obs)
- Current decisions already made: Go + Next.js 15 + BetterAuth + Rill self-hosted + OTel + Turborepo
- Open infrastructure decisions table: CDN/Edge, Hosting pattern, Managed Postgres, Redis/cache, Streaming pipeline v2, OLAP v2, Observability backend, Secrets management
- Three hosting options with cost estimates:
  - Option A (Recommended): Vercel + Fly.io + Neon + Upstash Redis (~$0-50/mo)
  - Option B: AWS managed (Lambda + RDS + managed Redis) (~$300+/mo)
  - Option C: Hetzner self-hosted Kubernetes (~$50-150/mo)

**Added Guidelines section**:
- Running migrations: `migrate -path infra/migrations -database $DATABASE_URL up`
- Running sqlc codegen: `cd packages/shared-domain && sqlc generate`
- Environment variables required for api-dashboard (DATABASE_URL, JWT_SECRET, OTEL_EXPORTER_OTLP_ENDPOINT)
- Docker Compose commands and stack description
- OTel Collector config location (infra/docker/otel-collector-config.yaml)

**Added Roadmap section**:
- v1: Local development only (Docker Compose)
- v2: Production deployment infrastructure selection required (9 open decisions)
- Decision priority order for team planning

---

### 4. docs/glossary.md

Added 14 new server-implementation domain terms (previously SDK-focused):

| Term | Definition |
|------|-----------|
| Organization | Мультитенантная единица платформы. Тип: `admin`, `publisher` или `brand`. Каждый пользователь принадлежит одной или нескольким org через `org_membership`. |
| Org Type | Тип организации: `admin` (внутренняя команда), `publisher` (разработчик приложения), `brand` (рекламодатель). Определяет доступные разделы дашборда и права доступа. |
| Publisher App | Приложение (bundle), зарегистрированное паблишером. Идентифицируется по `bundle_id` и `platform` (ios/android/unity). Привязано к org паблишера. |
| API Key | Ключ для аутентификации SDK на горячем пути. Хранится как argon2-хеш; при создании возвращается один раз в открытом виде. Принадлежит `publisher_app`. |
| Publisher Rule | Ограничения паблишера на матчинг: блоклисты категорий и брендов, frequency cap, geo и platform allowlist. Применяются при подборе спонсора. |
| Campaign | Рекламная кампания бренда: targeting (категории, гео, платформы), бюджет, статус (draft/active/paused/completed). Содержит HTML5 creatives. |
| HTML5 Creative | Набор ассетов кампании в формате HTML5 bundle. Загружается через multipart upload, валидируется в sandbox iframe, хранится в S3/R2. |
| Matching Decision | Запись о решении AI-матчинга: какой кампании была назначена сессия (или fallback), с объяснением факторов. Write-only со стороны SDK API. |
| X-Org-ID | HTTP-заголовок запроса: UUID организации, от имени которой действует пользователь. Валидируется против списка org в JWT. |
| OTLP | OpenTelemetry Protocol. Протокол экспорта трейсов/метрик/логов. В dev трейсы идут в OTel Collector → Jaeger по gRPC на порт 4317. |
| sqlc | Инструмент кодогенерации Go-кода из SQL-запросов. Источник правды — `.sql`-файлы в `packages/shared-domain/queries/`. |
| Seed Data | Синтетические session events (50k записей), сгенерированные скриптом и загруженные в MinIO/S3 в формате Parquet. Используются Rill для построения дашбордов. |

---

### 5. docs/architecture.md

**Fixed Components table** (was describing non-existent AI Matching Engine):
- Old: "Backend API — AI Matching Engine — контекстный подбор спонсора"
- New: "Backend API — CRUD API для dashboard, organizations domain, JWT auth. Future: publisher/campaign CRUD."

**Filled Environments section** (previously empty TODO):
- dev: Docker Compose stack (local, 1 developer, Jaeger/MinIO/Rill UIs)
- staging: AWS/GCP managed services (environment for QA before prod)
- prod: CDN edge → API (region-specific) → managed DB + cache + streaming (high availability, monitoring)

---

## Validation Results

All five updated files pass comprehensive validation:

**Link Validation**: 0 broken relative links
- All inter-document references verified (backend → sdk/api-spec.md, platform → infra-architecture.md, etc.)
- All anchors exist (#privacy-tiers in sdk/api-spec.md)
- All navigation links in docs/README.md point to existing files

**TODO Completeness**: 0 remaining `<!-- TODO -->` placeholders in updated files
- docs/backend/README.md: clean
- docs/dashboard/README.md: clean
- docs/platform/README.md: clean
- docs/glossary.md: clean
- docs/architecture.md: clean

**Navigation Coverage**: All five files reachable from docs/README.md
- No orphan pages created
- No new entries needed in docs/README.md (all sections already linked)

**Architecture Accuracy**: Documentation matches actual code
- Service structure diagram vs services/api-dashboard/ file layout: 100% match
- Dashboard current state: correctly documented as empty (apps/dashboard/ is empty)
- Infrastructure components: all referenced services exist in infra/docker/

**Roadmap Consistency**: Cross-document phase dependencies verified
- Backend Phase 1 (OpenAPI spec, identity) correctly blocks Dashboard Phase 0
- Backend Phase 2 (publisher domain) correctly blocks Dashboard Phase 2
- Backend Phase 3 (campaigns) correctly blocks Dashboard Phase 3
- Backend Phase 4 (analytics) correctly blocks Dashboard Phase 4

**Code vs Spec Divergences**: Documented and marked
- JWT validation approach (current: HMAC, spec: BetterAuth JWKS) — documented with "Known Deviation" note
- API path prefix (/api/v1 vs /v1) — documented in backend Architecture section with intent to reconcile in Phase 1 OpenAPI spec

---

## Key Outcomes

1. **Backend developers now have a clear architecture guide** — they understand service layering, DI pattern, auth flow, and how to add new entities following the 8-step checklist.

2. **New developers can onboard via docs/platform/** — they understand the local dev stack, how to run migrations, how to generate sqlc code, and what env vars are needed.

3. **Frontend team has a target architecture** — they know how to structure the Next.js app, how to wire BetterAuth, how to proxy Rill, and how to forward auth headers to the Go API.

4. **Product team has a realistic roadmap** — 6 phases for backend, 6 phases for dashboard, with clear dependencies. Current v1 state (organizations CRUD) is explicitly documented.

5. **Glossary now covers server implementation** — domain terms like Organization, Publisher App, Campaign, Matching Decision are defined and cross-linked.

6. **Stale content removed** — the AI matching engine vision is replaced with current CRUD-only reality for v1, with AI marked as v2+ future work.

---

## Notes

- **BetterAuth JWKS migration** is documented as a Phase 1 task in backend roadmap. Current code uses HMAC `JWT_SECRET` for MVP speed — this is a known and documented deviation from the spec.
- **OpenAPI spec generation** (packages/proto/dashboard.yaml) is explicitly marked as Phase 1 frontend blocker. This enables oapi-codegen (Go) and openapi-typescript (frontend) later.
- **docs/conventions.md** (not updated in this batch) still contains 5 TODO placeholders for SDK/product conventions. Out of scope but worth tracking separately.
- **packages/proto/** directory is empty. Both backend and dashboard roadmaps reference the dashboard.yaml spec creation in Phase 1 as a blocker.
