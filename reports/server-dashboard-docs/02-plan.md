# Plan: Обновление docs/backend, docs/dashboard, docs/platform

## Что делаем

Заполняем 11 TODO-секций в 3 документах + фиксим stale контент + обновляем glossary.

---

## docs/backend/README.md

### Overview (FIX stale)
- Убрать описание AI Matching Engine, Continuous Learning — их нет и не будет в v1
- Написать актуальный overview: multi-tenant CRUD API для dashboard, organizations + будущие publisher/brand domains
- Оставить ссылку на SDK API spec как future reference

### Architecture
- Layered structure: handler → service → repository → sqlc (с диаграммой)
- DI pattern (main.go wiring chain)
- JWT auth flow (текущий: HMAC; planned: BetterAuth JWKS) — пометить deviation
- Middleware chain: otelchi → RequestID → RealIP → Recoverer → ValidateJWT → RequireRole
- Response envelope format
- Env vars таблица
- Route table (текущие endpoints)
- Data flow diagram (request → handler → service → repo → DB)

### Guidelines
- Error handling pattern (sentinel errors → wrapping → handler mapping)
- Testing pattern (mock interface, noop tracer, table-driven)
- sqlc usage (queries → generate → wrap in repo)
- New entity checklist (8 steps)

### Roadmap (фазы + задачи)

**Phase 0: Foundation (DONE)**
- Monorepo skeleton, docker-compose, Rill + seed, organizations CRUD, JWT middleware, OTel tracing

**Phase 1: OpenAPI + Identity**
- packages/proto/dashboard.yaml — OpenAPI 3.1 spec
- oapi-codegen setup + Makefile target
- Reconcile path prefix: /api/v1 vs /v1
- Migration: users + org_memberships
- sqlc queries + model + repo + service + handler для users
- GET /v1/me endpoint
- BetterAuth JWKS validation (migrate from HMAC)

**Phase 2: Publisher Domain**
- Migration: publisher_apps, api_keys, publisher_rules
- Full CRUD: publisher-apps (list, create, get)
- API key provisioning (argon2 hash, plaintext return once)
- API key revocation
- Publisher rules CRUD (get, update)
- Pagination на list endpoints

**Phase 3: Brand/Campaign Domain**
- Migration: campaigns, creatives
- Campaign CRUD (list, create, get, status patch)
- Creative upload: multipart → S3/MinIO → HTML5 validation
- Creative preview (sandboxed iframe URL)

**Phase 4: Analytics + Embeds**
- Rill embed signed URL endpoint
- DuckDB aggregation endpoints (publisher metrics, campaign performance)
- Matching decision table migration (read-only from dashboard)

**Phase 5: Observability Polish**
- OTel metrics (request rate/latency/error per endpoint)
- DB pool metrics
- Structured log correlation (trace_id in logs)

**Future (v2+)**
- api-sdk hot-path service
- AI matching model (replace rules stub)
- Real-time event pipeline (Redpanda/NATS)
- Billing (Stripe)

---

## docs/dashboard/README.md

### Overview (FIX stale)
- Убрать конкретные метрики (94.2% fill rate, $18.40 RPM) — это product vision
- Переписать как planned architecture + текущий статус (empty)

### Architecture
- Next.js 15 App Router + React 19 + TypeScript
- BetterAuth self-hosted (auth endpoints, session, JWT issuance)
- shadcn/ui + Tailwind v4
- TanStack Query + TanStack Table
- Recharts для custom analytics (user-facing)
- Rill iframe proxy pattern (proxy через Next.js, signed URL, org-scoped)
- Page structure по ролям (publisher/brand/admin)
- API client: openapi-typescript generated → openapi-fetch
- JWT forwarding: Next.js → Go API (Authorization + X-Org-ID headers)

### Guidelines
- Component naming (PascalCase), hooks (use prefix), types (no I prefix)
- Auth pattern в Next.js (Server Components → auth(), Client → session hook)
- API client usage pattern
- Rill iframe security

### Roadmap (фазы + задачи)

**Phase 0: Foundation (NOT STARTED)**
- Next.js 15 scaffold + BetterAuth setup
- shadcn/ui + Tailwind v4 init
- Layout: sidebar + topbar + org switcher
- openapi-typescript client codegen setup

**Phase 1: Auth Pages**
- /login, /signup
- /accept-invite/:token
- /onboarding (create org, first app/campaign)

**Phase 2: Publisher Pages**
- /apps — list, detail, API key management, rules editor

**Phase 3: Brand Pages**
- /campaigns — list, detail, creative upload + preview

**Phase 4: Analytics Pages**
- /analytics — Rill iframe (publisher-view, brand-view)
- Custom Recharts charts for key metrics

**Phase 5: Admin**
- /admin/* — cross-org overview

**Future (v2+)**
- Real-time session dashboard
- ML matching transparency UI
- Billing/payouts UI

---

## docs/platform/README.md

### Overview
- Local dev stack: Docker Compose (Postgres 17, MinIO, Rill, OTel Collector, Jaeger)
- Make commands таблица (infra-up, infra-down, seed, rill-ui, jaeger-ui, minio-ui)
- Default ports таблица
- Prerequisites (Go 1.23+, sqlc, golang-migrate, Docker, pnpm 9.15+, Node 20+)

### Architecture
- Target infrastructure diagram (из infra-architecture.md, Mermaid)
- 12 layers overview
- Decisions already made (таблица)
- Open decisions (таблица с TBD)
- 3 hosting options (A/B/C) с cost estimates

### Guidelines
- Running migrations: `migrate -path infra/migrations -database $DATABASE_URL up`
- Running sqlc codegen: `cd packages/shared-domain && sqlc generate`
- Env vars required for api-dashboard
- Docker Compose commands
- OTel Collector config location

### Roadmap
- Open infra decisions (9 пунктов из infra-architecture.md)
- Decision priority order
- v1: local dev only (docker-compose)
- v2: prod deployment (decision needed)

---

## docs/glossary.md — добавить 11 терминов

Organization, Org Type, Publisher App, API Key, Publisher Rule, Campaign, HTML5 Creative, Matching Decision, X-Org-ID, OTLP, sqlc, Seed Data

---

## docs/architecture.md — fix stale

- Components table: Backend API = "CRUD API для dashboard" (не AI Matching Engine)
- Environments TODO: описать dev (docker-compose), staging (TBD), prod (TBD)

---

## Порядок работы

1. docs/backend/README.md (highest priority — блокирует разработку)
2. docs/platform/README.md (блокирует onboarding нового разработчика)
3. docs/dashboard/README.md (блокирует фронтенд разработку)
4. docs/glossary.md (дополнение)
5. docs/architecture.md (fix stale)
