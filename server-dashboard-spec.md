# BrandMoment: Server + Dashboard — Specification

**Версия:** v1 (first release)
**Дата:** 2026-04-14
**Автор:** discovery interview (Claude + Denis)

---

## 1. Executive Summary

Поднимаем с нуля **admin + self-serve dashboard** и **Dashboard API** платформы BrandMoment. В этой итерации фокус на **многотенантном dashboard** с CRUD для Publishers / Campaigns / Publisher Rules и **embedded-аналитикой на Rill** (seed-данные в S3/Parquet). SDK hot-path (`/v1/session/*`) — отдельный сервис, строится в следующей итерации, но архитектура репо и данных готовится под него сейчас.

**Принципы:**
- 1 разработчик + AI, без жёстких дедлайнов → приоритет на масштабируемую архитектуру, а не на скорость.
- Микросервисы с первого дня (dashboard-API и будущий SDK-API — разные профили нагрузки).
- Контроль данных и отсутствие vendor lock-in там, где возможно (BetterAuth, Rill self-hosted).

---

## 2. Goals / Non-Goals

### Goals (v1)
- Multi-tenant dashboard с orgs: **admin**, **publisher**, **brand**.
- CRUD для всех четырёх сущностей: Publishers + apps/bundles + API keys, Campaigns, Publisher Rules, Analytics-пан­ели (Rill-embed).
- HTML5 creatives upload + preview.
- Full observability с нуля (OpenTelemetry traces/metrics/logs).
- Repo и data layer спроектированы под добавление SDK API без переделки.

### Non-Goals (v1)
- SDK hot-path API (`/v1/sdk/init`, `/v1/session/request`, `/v1/session/event`) — отдельная итерация.
- Billing и publisher payouts (Stripe) — не в MVP.
- Static/video creatives — только HTML5 в v1.
- Russian / i18n — только English.
- Enterprise SSO (SAML/SCIM) — BetterAuth social/email login достаточно.
- ML-модель AI matching — в v1 matching это stub/rules-driven; ML-часть в будущих итерациях.

---

## 3. User Stories

**Admin (internal team):**
- Импорт/создание orgs вручную, выдача ролей, мониторинг всех кампаний и паблишеров.

**Publisher:**
- Регистрируется → создаёт organization → добавляет app (bundle ID + platform) → получает `X-BM-Api-Key` → задаёт rules (category/brand blocklist, frequency cap, geo/platform filter) → видит Rill-дашборд со своими метриками (fill rate, RPM, retention impact).

**Brand:**
- Регистрируется → создаёт organization → создаёт campaign (targeting, budget placeholder, HTML5 creatives) → запускает → видит Rill-дашборд с campaign performance (delivery, sponsor visibility, audience breakouts, AI transparency).

---

## 4. Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Next.js 15 Dashboard                     │
│   (UI, BetterAuth endpoints, Rill iframe proxy, RBAC)      │
└──────┬──────────────────────────────────┬───────────────────┘
       │ JWT (BetterAuth)                 │ signed iframe URL
       ▼                                  ▼
┌─────────────────┐               ┌─────────────────────┐
│ Go api-dashboard│               │ Rill Developer      │
│ (chi/Fiber)     │               │ (Docker, self-host) │
│ CRUD + queries  │               │ reads Parquet in S3 │
└──────┬──────────┘               └──────────┬──────────┘
       │                                     │
       ▼                                     ▼
┌─────────────────┐               ┌─────────────────────┐
│ Postgres (OLTP) │               │ S3/R2 (Parquet)     │
│ orgs/users/apps │               │ session events      │
│ campaigns/rules │               │ (seed in v1)        │
│ creatives meta  │               │                     │
└─────────────────┘               └─────────────────────┘

[Future] services/api-sdk (Go) ──writes events──► S3/R2
```

**Ключевые решения:**
- **Go** для API: предсказуемая latency под будущие SDK SLA (800 ms), хорошая интеграция с ClickHouse/DuckDB/S3.
- **Next.js 15 App Router + React 19**: SSR/RSC, single deployment UI.
- **BetterAuth** (TS, self-hosted в Next.js): orgs, RBAC, invites, magic link, OAuth; выпускает JWT для Go API; без vendor lock-in.
- **Rill Developer self-hosted (Docker)**: читает Parquet из S3 через встроенный DuckDB. Embed через iframe за нашим reverse proxy с подписанными URL.
- **OpenTelemetry** — traces/metrics/logs с первого дня.

---

## 5. Repo Layout (Turborepo monorepo)

```
brandmoment-server/
├── services/
│   ├── api-dashboard/         # Go: REST API для dashboard (CRUD, auth, queries)
│   └── api-sdk/               # [Future] Go: hot-path для SDK
├── apps/
│   └── dashboard/             # Next.js 15 UI + BetterAuth endpoints
├── packages/
│   ├── proto/                 # OpenAPI 3.1 specs → gen Go + TS clients
│   └── shared-domain/         # Go: общие domain models, DB queries (sqlc)
├── infra/
│   ├── docker/                # docker-compose для dev (pg, rill, otel-collector, minio)
│   ├── migrations/            # Postgres migrations (golang-migrate)
│   └── terraform/             # [Future] hosting
├── docs/                      # submodule brandmoment-docs
├── turbo.json
├── pnpm-workspace.yaml
└── go.work
```

**Shared contract**: OpenAPI 3.1 в `packages/proto/` → кодоген:
- Go server stubs (`oapi-codegen`)
- TypeScript client (`openapi-typescript` + `openapi-fetch`) для dashboard UI.

---

## 6. Tech Stack (окончательный)

| Слой | Выбор |
|---|---|
| Backend API | **Go** (chi router + sqlc + pgx + otelgo) |
| Dashboard UI | **Next.js 15 App Router + React 19 + TypeScript** |
| Styling / UI kit | **Tailwind v4 + shadcn/ui** |
| Data fetching | **TanStack Query + TanStack Table** |
| Auth | **BetterAuth** (self-hosted в Next.js) + JWT для Go API |
| OLTP DB | **Postgres** (через `sqlc` + `pgx`) |
| Analytics store | **S3/R2 + Parquet**, читается через **DuckDB** (Go API) |
| Analytics charts | **Recharts / Tremor** в Next.js (кастомный UI для publishers/brands) |
| Internal BI | **Rill Developer** (Docker self-host, для внутренней аналитики, не user-facing) |
| Observability | **OpenTelemetry** (traces/metrics/logs) → бэкенд TBD (Jaeger/Tempo или Grafana Cloud) |
| Migrations | **golang-migrate** |
| API contract | **OpenAPI 3.1** (source of truth) |
| Monorepo | **Turborepo + pnpm + go.work** |

---

## 7. Data Models (Postgres, OLTP)

### 7.1 Identity / Tenancy
```
organization        id, type (publisher|brand|admin), name, slug, created_at
user                id, email, name, created_at
org_membership      id, org_id, user_id, role (owner|admin|editor|viewer)
```
RBAC enforcement — в Go middleware (проверка org_id в JWT claims + role в org_membership).

### 7.2 Publisher Domain
```
publisher_app       id, org_id (publisher), name, bundle_id, platform (ios|android|unity), created_at
api_key             id, publisher_app_id, key_hash, key_prefix, name, last_used_at, revoked_at
publisher_rule      id, org_id (publisher), scope (global|per_app), app_id?,
                    category_blocklist text[], brand_blocklist uuid[],
                    frequency_cap_per_day int, geo_allowlist text[], platform_allowlist text[]
```
API key: храним `argon2(hash)` + `prefix` (первые 8 символов для UI); валидация на SDK hot-path будет читать из Redis-кеша + Postgres fallback.

### 7.3 Brand / Campaign Domain
```
campaign            id, org_id (brand), name, status (draft|active|paused|completed),
                    targeting jsonb (categories, geos, platforms, flights),
                    budget_placeholder numeric, created_at
creative            id, campaign_id, type (html5), storage_key (S3), entry_html_path,
                    dimensions, checksum, validated_at
```

### 7.4 Matching Decisions (write-only audit)
```
matching_decision   id, session_id, publisher_app_id, campaign_id?, decision (match|fallback),
                    reasons jsonb (AI transparency), created_at
```
Пишется SDK API (будущий), читается dashboard API для explainability UI.

---

## 8. Event Pipeline (для Rill)

### v1 (seed)
- Скрипт `infra/seed/generate-events.go` генерирует реалистичные session events → Parquet → загружает в S3/MinIO.
- Rill в Docker читает S3 (DuckDB httpfs extension) и строит metrics views:
  - `session_events` (session_id, publisher_app_id, campaign_id, platform, geo, duration_ms, ack_shown, badge_shown, outcome, rpm_cents, ts)
  - Публикует dashboards: publisher-view, brand-view, admin-view.

### v2+ (real)
- SDK API пишет события батчами в Kafka / Kinesis / прямо в S3 (выбор позже).
- Rotating Parquet writer → `s3://brandmoment-events/year=YYYY/month=MM/day=DD/hour=HH/*.parquet`.
- Rill реагирует на новые партиции автоматически.

---

## 9. API Contracts (dashboard API)

Source of truth: `packages/proto/dashboard.yaml` (OpenAPI 3.1). Резюме по ресурсам:

```
POST   /v1/auth/session/exchange     # BetterAuth JWT → server JWT (если нужен свой issuer)
GET    /v1/me

# Organizations (multi-tenant)
GET    /v1/orgs/:id
POST   /v1/orgs/:id/invites
...

# Publisher Apps
GET    /v1/publisher-apps
POST   /v1/publisher-apps
POST   /v1/publisher-apps/:id/api-keys
DELETE /v1/api-keys/:id

# Publisher Rules
GET    /v1/publisher-rules
PUT    /v1/publisher-rules/:id

# Campaigns (brand side)
GET    /v1/campaigns
POST   /v1/campaigns
POST   /v1/campaigns/:id/creatives      # multipart → S3 → валидация HTML5 bundle
PATCH  /v1/campaigns/:id/status

# Embeds
GET    /v1/embeds/rill/signed-url?view=publisher|brand&org_id=...
```

Ответы единообразны: `{ data, error: null } | { data: null, error: { code, message, details } }`.

---

## 10. UI / UX Requirements

- **Design system**: shadcn/ui + Tailwind v4, dark mode из коробки.
- **Layout**: sidebar (по типу org показываются разные разделы) + top bar (org switcher, user menu).
- **Страницы v1**:
  - `/login`, `/signup`, `/accept-invite/:token`
  - `/onboarding` (создание org, первый app/campaign)
  - `/apps` (publisher): список, detail, API keys, rules
  - `/campaigns` (brand): список, detail, creatives upload, preview
  - `/analytics` (обе роли): Rill iframe с разными views
  - `/admin/*` (admin org): полный обзор всех orgs
- **Loading/error states**: skeleton UI + toast на ошибках (sonner).
- **Accessibility**: shadcn/ui по умолчанию WCAG AA; focus states обязательно.

---

## 11. Security Considerations

- **API keys** (публичные, в SDK) — argon2-хеш в БД, возвращаются plaintext один раз при создании. Префикс для UI.
- **BetterAuth JWT** — короткоживущие (15 мин) + refresh token (14 дней), JWKS endpoint для Go-валидации.
- **RBAC**: проверка `org_id + role` на каждом endpoint (middleware). Row-level фильтрация по `org_id` во всех queries.
- **HTML5 creatives**: валидация при upload (sandboxed preview: `<iframe sandbox="allow-scripts">`, CSP strict, запрет `top` navigation, размер лимит, whitelist URL-паттернов). Храним в S3 с pre-signed URL.
- **Rill embed**: proxy через наш Next.js → проверка session → подписанный URL на Rill с ограниченным scope (org-level filter). Никогда не светим Rill напрямую.
- **Secrets**: открытый вопрос (см. §14).
- **CSRF**: BetterAuth session cookies SameSite=Lax; JWT for API calls.

---

## 12. Observability

С первого дня:
- **Traces**: OpenTelemetry в Go API и Next.js (propagation через W3C traceparent).
- **Metrics**: OTLP → Prometheus-compatible backend. Обязательные метрики: request rate/latency/error per endpoint, DB pool, Rill embed load time.
- **Logs**: structured (JSON, `slog` в Go, `pino` в Node), с trace_id/span_id.
- **Local dev**: otel-collector + Jaeger/Tempo в docker-compose.
- **Prod backend**: выбор позже (Grafana Cloud / SigNoz self-hosted / Honeycomb).

---

## 13. Risks & Mitigations

| Риск | Вероятность | Impact | Митигация |
|---|---|---|---|
| HTML5 creatives → XSS в dashboard preview | Высокая | Высокий | Strict sandboxed iframe, CSP, загрузка на отдельный домен (`creatives.brandmoment.dev`) |
| BetterAuth ограничен в зрелости vs Clerk/WorkOS | Средняя | Средний | Изоляция auth за тонким интерфейсом; план на миграцию на WorkOS при enterprise-клиентах |
| Rill embed security (leak данных между orgs) | Средняя | Высокий | Scope через signed URL + org-level фильтры в Rill metrics view |
| Parquet-seed данные оторвутся от реальной схемы SDK | Средняя | Средний | Схема событий — в `packages/proto/events.yaml`, seed и SDK строятся от одного источника |
| Monorepo overhead для одного разработчика | Низкая | Низкий | Turborepo кеш + чёткие границы между packages |
| S3 egress cost при Rill запросах | Низкая | Низкий | Использовать Cloudflare R2 (zero egress) или локальный кеш Parquet в volume |

---

## 14. Open Questions

1. **Hosting** — где деплоим prod? Варианты: Vercel+Fly.io+Neon / AWS / Hetzner k8s. **Не решено.**
2. **Observability backend** — Grafana Cloud / SigNoz / Honeycomb. **Не решено.**
3. **Secrets management** — `.env` (dev) ок, а для prod Doppler / AWS SM / Infisical? **Не решено.**
4. **Event ingest pipeline** (когда дойдём до SDK) — Kafka / Kinesis / Redpanda / прямые batched uploads в S3? **Не решено.**
5. **Need for internal session JWT?** — или достаточно BetterAuth-JWT → прямое потребление в Go? **Склоняюсь к "достаточно".**
6. **Rill RBAC модель** — одна Rill instance с org-фильтрами через params или инстанс-на-org? **Склоняюсь к одной с фильтрами.**

---

## 15. Success Criteria (v1 complete)

- [ ] Любой новый пользователь может зарегистрироваться, создать publisher- или brand-org, пригласить коллегу.
- [ ] Publisher создаёт app, получает API key, настраивает правила (все 4 типа).
- [ ] Brand создаёт campaign, грузит HTML5-creative, видит preview.
- [ ] Обе роли открывают `/analytics` и видят live Rill-дашборд на seed-данных с фильтром по их org.
- [ ] Admin видит cross-org обзор.
- [ ] Все API покрыты OpenAPI, Go + TS клиенты сгенерированы.
- [ ] Traces + metrics + logs идут в otel-collector локально; прошли e2e тест полного flow с trace-id корреляцией.
- [ ] CI: lint + test + build в Turborepo проходят.

---

## 16. Out of Scope (v1, зафиксировано)

- SDK hot-path API (`/v1/sdk/init`, `/v1/session/*`)
- AI matching model (stub / rules-based placeholder в v1)
- Real-time ingestion pipeline (Kafka/Kinesis)
- Static и video creatives
- Billing (Stripe) и publisher payouts (Stripe Connect)
- Russian / другие локали
- Enterprise SSO (SAML/SCIM)
- Public landing site (отдельный проект, см. `docs/landing/`)
- Mobile app reference implementations (`docs/android/`, `docs/ios/`, `docs/unity/`)

---

## 17. Next Steps (suggested)

1. ~~Скелет monorepo: `turbo.json`, `pnpm-workspace.yaml`, `go.work`, `infra/docker/docker-compose.yml`~~ ✅ DONE
2. ~~Rill Docker + seed data generator (50k events Parquet) + publisher/campaign dashboards~~ ✅ DONE
3. `packages/proto/dashboard.yaml` — OpenAPI 3.1 первая версия (orgs, users, publisher-apps).
4. `services/api-dashboard/` — health endpoint, otel wiring, pg connection, первая миграция (identity/tenancy).
5. `apps/dashboard/` — Next.js 15 skeleton + BetterAuth setup + shadcn/ui.
6. JWT validation bridge: BetterAuth JWKS → Go middleware.
7. Go API: DuckDB-агрегации из Parquet → REST endpoints для графиков (publisher metrics, campaign performance).
8. Next.js: кастомные графики (Recharts/Tremor) для publishers/brands.
9. CRUD для publisher-apps + api-keys → первый E2E "signup → create app → get key".
