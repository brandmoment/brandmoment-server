# Task Report: Organizations CRUD

## Задача

Создать полный CRUD для Organizations в сервисе `api-dashboard` — от SQL миграции до HTTP endpoints с auth, tracing и тестами.

---

## Какие правила из CLAUDE.md были применены

### Pre-flight Checks
Перед генерацией кода проверил наличие инструментов. `go`, `sqlc`, `migrate` отсутствовали — остановился и запросил установку. Только после `brew install` продолжил работу.

### New Entity Checklist
Строго следовал порядку из CLAUDE.md:
1. SQL migration → 2. sqlc queries → 3. Model → 4. Repository → 5. Service → 6. Handler → 7. Tests → 8. Router → 9. Post-generation checks

### Multi-Tenancy Model
- Organizations — **top-level resource**, НЕ sub-resource. Не добавлял `org_id` колонку и `WHERE org_id = $1` в запросы.
- Доступ контролируется через JWT membership: `ListOrganizationsByIDs` возвращает только те орги, в которых пользователь состоит (`orgs[]` claim).
- `X-Org-ID` header валидируется против JWT claims.

### Auth: BetterAuth
- Использовал `golang-jwt/jwt/v5` для валидации токенов (не ручной парсинг).
- RBAC middleware (`RequireRole`) проверяет роль из `orgs[]` массива JWT.
- `org_id` берётся из контекста, а не из тела запроса.

### Naming Conventions
| Правило | Применение |
|---------|------------|
| Packages: lowercase, no underscores | `httputil`, `middleware`, `repository` |
| Files: snake_case.go | `organization.go`, `organization_test.go` |
| Types: PascalCase + suffix | `OrganizationService`, `OrganizationRepository`, `OrganizationHandler` |
| Constructors: `New...` | `NewOrganizationService(deps)` |
| Errors: `Err...` | `ErrNotFound`, `ErrUnauthorized`, `ErrInvalidInput` |
| Tests: table-driven | `TestOrganizationService_Create` |
| Tables: snake_case, plural | `organizations` |
| PKs: `id UUID` | `id UUID PRIMARY KEY DEFAULT gen_random_uuid()` |
| Timestamps: TIMESTAMPTZ | `created_at TIMESTAMPTZ NOT NULL DEFAULT now()` |
| Enums: TEXT + CHECK | `CHECK (type IN ('admin', 'publisher', 'brand'))` |
| Endpoints: kebab-case, versioned | `/api/v1/organizations` |
| Response envelope | `{"data": ..., "error": {"code": "...", "message": "..."}}` |

### SQL Conventions
- Обе миграции: `.up.sql` + `.down.sql`
- `IF NOT EXISTS` / `IF EXISTS` для идемпотентности
- Индекс на `slug` (используется в WHERE)
- Нумерация: `000001`

### Go Backend Rules
- Router вынесен в `internal/router/router.go` как `NewRouter()` — не inline в main
- Handler: decode → delegate → respond (без бизнес-логики)
- Service: DI + OTel tracing + slog (не fmt.Println)
- Repository: wraps sqlc, never raw SQL
- Response helpers: единый источник (`httputil` пакет), middleware импортирует оттуда
- Import order: stdlib → third-party → internal

### Anti-Patterns (avoided)
- NO global state / init()
- NO panics в бизнес-логике
- NO raw SQL в handler/service/repository
- NO fmt.Println — только slog
- NO custom JWT parsing — golang-jwt/jwt/v5
- NO duplicate response helpers

### Post-generation Checks
- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./...` — OK (9 tests passed)

---

## Какие Tools использовал

| Tool | Для чего | Количество вызовов |
|------|----------|-------------------|
| **Read** | Чтение существующих файлов (go.mod, go.work, sqlc-generated code) | 6 |
| **Write** | Создание новых файлов (миграции, Go код, конфиги) | 15 |
| **Edit** | Точечное редактирование go.mod (добавление replace directive) | 1 |
| **Bash** | Установка brew пакетов, go mod tidy, go build, go vet, go test, mkdir, sqlc generate | 14 |
| **Glob** | — (не потребовался, структура известна из Explore) | 0 |
| **Grep** | — (не потребовался) | 0 |
| **Agent (Explore)** | Исследование структуры репозитория перед началом работы | 1 |
| **TaskCreate** | Создание 11 задач по Entity Checklist | 11 |
| **TaskUpdate** | Отметки in_progress/completed по ходу работы | 18 |

---

## Какие Skills / Agent Types использовал

| Skill/Agent | Контекст |
|-------------|----------|
| **Agent (Explore)** | Первичное исследование codebase: что уже существует в `services/api-dashboard/`, `packages/shared-domain/`, `infra/migrations/` |
| **sqlc generate** | Кодогенерация Go-обёрток над SQL-запросами |
| **go mod tidy** | Автоматическое разрешение зависимостей |

Skills `/simplify` и `/security-review` рекомендованы к запуску как следующий шаг (Post-generation Checks из CLAUDE.md).

---

## Что было создано

```
infra/migrations/
├── 000001_create_organizations.up.sql
└── 000001_create_organizations.down.sql

packages/shared-domain/
├── sqlc.yaml
├── queries/organizations.sql
└── db/                          # sqlc-generated
    ├── db.go
    ├── models.go
    └── organizations.sql.go

services/api-dashboard/
├── cmd/server/main.go
├── go.mod
├── go.sum
└── internal/
    ├── config/config.go
    ├── httputil/response.go
    ├── middleware/auth.go
    ├── model/organization.go
    ├── repository/organization.go
    ├── service/
    │   ├── organization.go
    │   └── organization_test.go
    ├── handler/
    │   ├── organization.go
    │   └── health.go
    └── router/router.go
```

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/healthz` | No | Health check |
| POST | `/api/v1/organizations` | JWT + RBAC | Create organization |
| GET | `/api/v1/organizations` | JWT + RBAC | List user's organizations |
| GET | `/api/v1/organizations/{id}` | JWT + RBAC | Get organization by ID |
