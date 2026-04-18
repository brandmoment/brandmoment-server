# Code vs Spec: Полный анализ расхождений

Дата: 2026-04-18
Источники: `server-dashboard-spec.md`, `infra-architecture.md`, код `services/api-dashboard/`, `packages/shared-domain/`, `infra/`

---

## 1. Что реально построено

| Компонент | Статус | Детали |
|-----------|--------|--------|
| Monorepo skeleton | ✅ Done | Turborepo, go.work, pnpm-workspace |
| Docker Compose | ✅ Done | Postgres 17, MinIO, Rill, OTel Collector, Jaeger |
| Rill dashboards | ✅ Done | publisher_overview, campaign_performance + 50k seed events |
| Organizations CRUD | ✅ Done | Create, GetByID, List — полный стек handler→service→repo→sqlc |
| JWT auth middleware | ✅ Done | HMAC symmetric (не JWKS) |
| OTel tracing | ✅ Done | OTLP gRPC → Jaeger, per-method spans |
| Next.js dashboard | ❌ Empty | `apps/dashboard/` пуст — ни одного файла |
| OpenAPI spec | ❌ Empty | `packages/proto/` пуст — нет dashboard.yaml |

---

## 2. Расхождения код ↔ spec

### 2.1 JWT: HMAC vs BetterAuth JWKS

| | Spec (§4, §11) | Код |
|---|---|---|
| Метод | Asymmetric JWKS — валидация через BetterAuth JWKS endpoint | Symmetric HMAC — `JWT_SECRET` env var |
| Библиотека | `golang-jwt/jwt/v5` + JWKS Keyfunc | `golang-jwt/jwt/v5` + HMAC signing method |
| Production-ready? | Да | Нет — HMAC secret не масштабируется, нет ротации ключей |

**Impact:** текущая auth не совместима с BetterAuth token issuance. Нужен JWKS-based `Keyfunc`. Это Phase 1 задача.

### 2.2 Path prefix: `/api/v1` vs `/v1`

| | Spec (§9) | Код |
|---|---|---|
| Prefix | `/v1/orgs/:id` | `/api/v1/organizations/{id}` |
| Resource name | `orgs` (short) | `organizations` (full) |

**Impact:** когда появится OpenAPI spec — нужно выбрать один формат. Codegen (oapi-codegen, openapi-typescript) будет генерировать по spec. Если выберем spec-формат — нужен рефакторинг router.

### 2.3 Response envelope: omitempty vs explicit null

| | Spec (§9) | Код |
|---|---|---|
| Success | `{"data": ..., "error": null}` | `{"data": ...}` (error отсутствует) |
| Error | `{"data": null, "error": {...}}` | `{"error": {...}}` (data отсутствует) |

**Impact:** минимальный. Go `omitempty` убирает null-поля. TS client может ожидать explicit null. Решается при написании OpenAPI spec.

### 2.4 Organizations access pattern

| | Spec (§7.1) | Код |
|---|---|---|
| Access check | «через JWT membership» | Handler проверяет `slices.Contains(orgIDs, id)` **до** вызова service |

**Impact:** access check живёт в handler, а не в service. Работает правильно, но при добавлении новых entity стоит решить: access check в middleware (DRY) или в handler (explicit).

---

## 3. Data Models: что есть vs что нужно

Spec §7 определяет **9 таблиц**. Реализована **1**.

### Identity / Tenancy (§7.1)

| Таблица | Поля в spec | Статус | Блокирует |
|---------|-------------|--------|-----------|
| `organizations` | id, type, name, slug, created_at, updated_at | ✅ Migration 000001 | — |
| `users` | id, email, name, created_at | ❌ Нет | `/v1/me`, invites, всё остальное |
| `org_memberships` | id, org_id, user_id, role | ❌ Нет | RBAC в production (сейчас роли из JWT, нет DB) |

### Publisher Domain (§7.2)

| Таблица | Статус | Блокирует |
|---------|--------|-----------|
| `publisher_apps` | ❌ Нет | Регистрация приложений, API keys |
| `api_keys` | ❌ Нет | SDK интеграция, argon2 hash |
| `publisher_rules` | ❌ Нет | Matching rules engine |

### Brand / Campaign Domain (§7.3)

| Таблица | Статус | Блокирует |
|---------|--------|-----------|
| `campaigns` | ❌ Нет | Campaign management |
| `creatives` | ❌ Нет | HTML5 upload + preview |

### Matching (§7.4)

| Таблица | Статус | Блокирует |
|---------|--------|-----------|
| `matching_decisions` | ❌ Нет | AI transparency dashboard (read-only от dashboard) |

---

## 4. API Endpoints: что есть vs что нужно

Spec §9 определяет **~15 endpoint groups**. Реализовано **3** (partial).

| Endpoint | Spec | Код | Gap |
|----------|------|-----|-----|
| `GET /v1/me` | ✅ | ❌ | User profile |
| `POST /v1/auth/session/exchange` | Optional | ❌ | JWT exchange (может не понадобиться) |
| `GET /v1/orgs/:id` | ✅ | ⚠️ Partial | Есть как `/api/v1/organizations/{id}`, нет update/delete |
| `POST /v1/orgs/:id/invites` | ✅ | ❌ | Org invitations |
| `GET /v1/publisher-apps` | ✅ | ❌ | Publisher apps list |
| `POST /v1/publisher-apps` | ✅ | ❌ | Publisher app create |
| `POST /v1/publisher-apps/:id/api-keys` | ✅ | ❌ | API key provisioning |
| `DELETE /v1/api-keys/:id` | ✅ | ❌ | API key revocation |
| `GET /v1/publisher-rules` | ✅ | ❌ | Rules list |
| `PUT /v1/publisher-rules/:id` | ✅ | ❌ | Rules update |
| `GET /v1/campaigns` | ✅ | ❌ | Campaign list |
| `POST /v1/campaigns` | ✅ | ❌ | Campaign create |
| `POST /v1/campaigns/:id/creatives` | ✅ | ❌ | HTML5 upload → S3 |
| `PATCH /v1/campaigns/:id/status` | ✅ | ❌ | Status transitions |
| `GET /v1/embeds/rill/signed-url` | ✅ | ❌ | Rill embed proxy |

---

## 5. Observability gaps

| Что | Spec (§12) | Код | Gap |
|-----|-----------|-----|-----|
| Traces | ✅ OTel traces, W3C propagation | ✅ Done | — |
| Metrics | ✅ Request rate/latency/error per endpoint, DB pool | ❌ Нет | Нет counters/histograms |
| Logs | ✅ Structured JSON, trace_id/span_id | ⚠️ Partial | slog есть, но trace_id не коррелируется автоматически |
| Errors | ✅ Sentry | ❌ Нет | Нет Sentry integration |

---

## 6. Docs были stale

До обновления (уже пофикшено):

| Файл | Проблема |
|------|----------|
| `docs/backend/README.md` Overview | Описывал AI Matching Engine, Continuous Learning — ничего этого нет в v1 |
| `docs/architecture.md` Components | «Backend API = AI Matching Engine» — на самом деле CRUD API |
| `docs/glossary.md` | Только SDK/product термины, нет server-side (Organization, sqlc, OTLP) |
| 11 TODO секций | Пустые Architecture, Guidelines, Roadmap во всех 3 разделах |

---

## 7. Что делать дальше (приоритет)

### Критический путь (блокирует всё)

1. **OpenAPI spec** (`packages/proto/dashboard.yaml`) — без него нет codegen, нет contract
2. **Users + org_memberships** миграции — без них нет production auth
3. **BetterAuth JWKS** — заменить HMAC на asymmetric validation
4. **Path prefix decision** — `/api/v1` или `/v1`? Фиксируем в OpenAPI

### Publisher track (после identity)

5. `publisher_apps` + `api_keys` + `publisher_rules` — миграции + полный CRUD
6. Argon2 hashing для API keys

### Brand track (параллельно с publisher)

7. `campaigns` + `creatives` — миграции + CRUD
8. S3/MinIO upload + HTML5 validation

### Dashboard (блокируется backend Phase 1)

9. Next.js 15 scaffold + BetterAuth setup
10. openapi-typescript client codegen

### Polish

11. OTel metrics (histograms, counters)
12. Rill embed signed URL endpoint
13. DuckDB aggregation endpoints
