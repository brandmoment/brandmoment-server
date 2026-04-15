# Сравнение CLAUDE.md v1 vs v2

## Структура файлов

### v1
```
CLAUDE.md                    ← 380 строк, всё в одном файле
```

### v2
```
CLAUDE.md                    ← 190 строк (компактное ядро)
.claude/rules/
├── go-backend.md            ← Go паттерны (globs: **/*.go)
├── go-multi-tenancy.md      ← Multi-tenancy (globs: **/*.go)
└── sql-conventions.md       ← SQL/sqlc (globs: **/*.sql)
```

**Выигрыш**: rules загружаются только при работе с соответствующими файлами — экономия токенов.

---

## Что добавлено в v2

| Секция | v1 | v2 | Какую проблему решает |
|--------|----|----|----------------------|
| **Pre-flight Checks** | Нет | Таблица с `go version`, `sqlc version`, `docker --version` и т.д. | В генерации #1 Go не был установлен — узнали только в конце |
| **Post-generation Checks** | Нет | `go build`, `go vet`, `go test`, `/simplify`, `/security-review` | Код не проверялся после генерации |
| **New Entity Checklist** | Нет | Пошаговый чеклист: миграция → sqlc → model → repo → service → handler → tests → router | Ассистент не знал правильный порядок |
| **JWT Claims структура** | Нет | Пример `{"orgs": [{"org_id": "...", "role": "..."}]}` | Ассистент придумал упрощённый `{"org_id": "..."}` |
| **org_id filtering rules** | "All DB queries filtered by org_id" (одна строка) | Подробно: sub-resources vs top-level, как решать | Ассистент слепо добавил org_id к organizations — сломал SQL |
| **Auth: BetterAuth** | Нет | Запрет ручного JWT парсинга, использовать golang-jwt + JWKS | Ассистент написал ручной HMAC-SHA256 парсер |
| **RBAC middleware** | Только в примере router | Полная реализация RequireRole + правила безопасности | RBAC не был создан |
| **sqlc обязателен** | Упомянут в стеке | Anti-pattern: NO raw SQL в repository + пример sqlc.yaml + query files | Ассистент написал raw SQL через pool.Query |
| **Тесты обязательны** | Нет | Пример table-driven test, правило в чеклисте | 0 тестов в генерации #1 |
| **Router extraction** | Пример NewRouter() | Явное правило: router в отдельном файле, не в main.go | Router был inline в main.go |
| **Response helpers sharing** | Пример функций | Anti-pattern: NO duplicate response helpers, middleware импортирует из handler | middleware/auth.go создал свой writeAuthError |
| **go mod tidy** | Нет | Anti-pattern: NO guessing versions, всегда go mod tidy | go.mod без go.sum, версии "на глаз" |
| **Tools & Skills** | Нет | RTK, ast-index, JetBrains MCP, рекомендованные skills | Ни один skill не использовался |

---

## Что изменено в v2

| Секция | v1 | v2 |
|--------|----|----|
| **Multi-Tenancy** | 1 строка: "All DB queries filtered by org_id" | Подробная секция с примерами когда фильтровать, когда нет |
| **Tech Stack** | "API Contract: OpenAPI 3.1" | "Codegen: sqlc + oapi-codegen + openapi-typescript" |
| **Test naming** | `TestCreateCampaign` | `TestCampaignService_Create` (включает имя сервиса) |
| **Response format** | Отдельная секция с JSON | Одна строка в API conventions |
| **Code examples** | В CLAUDE.md (всегда грузятся) | В `.claude/rules/go-backend.md` (только при работе с Go) |
| **Anti-patterns** | В CLAUDE.md | В `.claude/rules/go-backend.md` + добавлены новые (raw SQL в repo, duplicate helpers, guessing versions) |

---

## Что убрано из CLAUDE.md в v2

Перенесено в `.claude/rules/`:
- Go Service Structure (дерево папок) → `go-backend.md`
- Good Code Examples (5 примеров) → `go-backend.md`
- Anti-Patterns (5 → 8 пунктов) → `go-backend.md`
- Typical Go File Template → `go-backend.md`
- API Response Format → перенесён в naming conventions (одна строка)

---

## Баги генерации #1 которые v2 должен предотвратить

| Баг | Корневая причина | Правило в v2 |
|-----|-----------------|--------------|
| `WHERE id = $1 AND id = $2` в repository | org_id применён к organizations слепо | "org_id filtering applies" — sub-resources vs top-level |
| Ручной HMAC JWT парсер | Нет указания какую библиотеку использовать | "Auth: BetterAuth" — используй golang-jwt + JWKS |
| JWT claims `{"org_id": "..."}` | Нет примера реальной структуры | JWT claims пример с массивом orgs |
| Raw SQL `pool.Exec` в repository | sqlc упомянут но не обязателен | Anti-pattern #4: NO raw SQL в repository implementations |
| Нет RBAC middleware | Нет явного требования | RequireRole пример + security rules |
| Нет тестов | Нет требования | "Tests" секция + entity checklist step 7 |
| Go не установлен | Нет pre-flight checks | Pre-flight Checks таблица |
| go.mod без go.sum | Нет post-flight checks | Anti-pattern #8 + Post-generation Checks |
| Router в main.go | Только пример, нет правила | "Router MUST be in internal/router/router.go" |
| writeAuthError дубликат | Нет правила про sharing | Anti-pattern #7: NO duplicate response helpers |

---

## Метрики правил

| Метрика | v1 | v2 |
|---------|----|----|
| Строк в CLAUDE.md | 380 | 190 |
| Файлов правил | 1 | 4 |
| Anti-patterns | 5 | 8 |
| Code examples | 5 | 5 (перенесены в rules) |
| Pre-flight checks | 0 | 6 инструментов |
| Post-generation checks | 0 | 5 шагов |
| Entity checklist | нет | 9 шагов |
| Tools/Skills описано | 0 | 4 (RTK, ast-index, JetBrains, skills) |

---

## Сравнение результатов генерации #1 (v1) и #2 (v2)

### Общая оценка

| Критерий | Генерация #1 (v1) | Генерация #2 (v2) |
|----------|--------------------|--------------------|
| Скомпилировалось? | Нет (Go не установлен, go.sum отсутствует) | Да (`go build ./...` OK) |
| Тесты? | 0 тестов | 9 тестов (3 table-driven), все прошли |
| go vet? | Не проверено | OK |
| Соответствует стилю? | ~70% | ~95% |

### Pre-flight Checks

| | v1 | v2 |
|---|---|---|
| Проверка инструментов | Нет. Обнаружил отсутствие Go только при попытке `go mod tidy` в конце | Да. Проверил `go`, `sqlc`, `migrate` в начале. Остановился, запросил `brew install` |
| Результат | go.mod написан вручную "на глаз", без go.sum | go.mod + go.sum сгенерированы через `go mod tidy` |

### Multi-Tenancy: Organizations

| | v1 | v2 |
|---|---|---|
| org_id в запросах | `WHERE id = $1 AND id = $2` — слепо применил org_id к organizations | Нет org_id фильтра. Organizations — top-level ресурс |
| Доступ к оргам | По org_id из JWT (одно поле) | Через `ListByIDs` + массив org_ids из JWT claims |
| SQL | `WHERE id = $1` в ListByOrg — возвращает max 1 строку | `WHERE id = ANY(@ids::uuid[])` — корректный запрос |

### Auth / JWT

| | v1 | v2 |
|---|---|---|
| JWT парсинг | Ручной HMAC-SHA256 (`crypto/hmac` + `encoding/base64`) | `golang-jwt/jwt/v5` библиотека |
| JWT claims | `{"org_id": "..."}` — одно поле | `{"orgs": [{"org_id": "...", "role": "..."}]}` — массив memberships |
| X-Org-ID header | Не реализован | Извлекается, валидируется против JWT claims |
| RBAC | Отсутствует | `RequireRole()` middleware, проверяет роль из claims |
| Контекст | Только `OrgIDFromContext()` | `OrgIDFromContext()` + `RoleFromContext()` + `OrgIDsFromContext()` |

### Repository

| | v1 | v2 |
|---|---|---|
| SQL | Raw SQL через `pool.Exec` / `pool.Query` | sqlc-generated `db.Queries` |
| sqlc config | Отсутствует | `packages/shared-domain/sqlc.yaml` |
| Query files | Отсутствуют | `packages/shared-domain/queries/organizations.sql` |
| sqlc generate | Не выполнялся | Выполнен, сгенерирован Go код |
| Интерфейс | `GetByID(ctx, orgID, id)` | `GetByID(ctx, id)` — без org_id (корректно для organizations) |

### Router

| | v1 | v2 |
|---|---|---|
| Расположение | Inline в `cmd/server/main.go` | Отдельный `internal/router/router.go` |
| RBAC на routes | Нет | `auth.RequireRole("viewer", "editor", "admin", "owner")` |
| Структура | Роутинг смешан с DI и server setup | Чистый `NewRouter(h, auth) http.Handler` |

### Response Helpers

| | v1 | v2 |
|---|---|---|
| Расположение | `handler/response.go` | `httputil/response.go` (отдельный пакет) |
| Дублирование | `writeAuthError()` в middleware — свой формат через `map[string]any` | Middleware импортирует `httputil.RespondError` — один источник |

### Тесты

| | v1 | v2 |
|---|---|---|
| Файлы тестов | 0 | 1 (`organization_test.go`) |
| Количество тестов | 0 | 9 (5 Create + 2 GetByID + 2 ListByIDs) |
| Паттерн | — | Table-driven, mock repository |
| Naming | — | `TestOrganizationService_Create` (по конвенции) |

### Tools & Skills использованные

| | v1 | v2 |
|---|---|---|
| Agent (Explore) | 1 | 1 |
| TaskCreate | 7 | 11 |
| Write | 12 | 15 |
| Bash | 6 | 14 (включая brew install, go mod tidy, go build, go test) |
| Skills | 0 | 0 (рекомендованы как next step) |
| sqlc generate | нет | да |
| go build | нет | да — прошёл |
| go vet | нет | да — прошёл |
| go test | нет | да — 9 passed |

### Файлы: v1 vs v2

| Файл | v1 | v2 |
|------|----|----|
| `infra/migrations/*.sql` | 2 файла | 2 файла |
| `packages/shared-domain/sqlc.yaml` | нет | да |
| `packages/shared-domain/queries/organizations.sql` | нет | да |
| `packages/shared-domain/db/*.go` | нет | да (sqlc-generated) |
| `internal/model/organization.go` | да | да |
| `internal/repository/organization.go` | raw SQL | wraps sqlc |
| `internal/service/organization.go` | да | да |
| `internal/service/organization_test.go` | нет | да (9 тестов) |
| `internal/handler/organization.go` | да | да |
| `internal/handler/response.go` | да | нет (перенесён в httputil) |
| `internal/httputil/response.go` | нет | да |
| `internal/handler/health.go` | да | да |
| `internal/middleware/auth.go` | ручной HMAC | golang-jwt/jwt/v5 |
| `internal/middleware/rbac.go` | нет | встроен в auth.go (RequireRole) |
| `internal/router/router.go` | нет (inline в main) | да |
| `internal/config/config.go` | да | да |
| `cmd/server/main.go` | да (с router inline) | да (чистый, router вынесен) |
| `go.mod` | без go.sum | + go.sum |

---

## Что больше всего повлияло на качество

1. **Pre-flight checks** — заставили ассистента установить Go перед генерацией. Код скомпилировался.
2. **"org_id filtering applies"** — явное правило когда фильтровать, когда нет. Исправлен критический баг с organizations.
3. **JWT claims пример** — ассистент не придумывал свою структуру, использовал реальный формат с массивом memberships.
4. **"NO custom JWT parsing"** — прямой запрет заставил использовать golang-jwt вместо ручного HMAC.
5. **sqlc как обязательный** — вместо raw SQL в repository, сгенерирован type-safe код.
6. **"Tests MUST"** — без этого правила тестов не было вообще. С ним — 9 table-driven тестов.
7. **"Router MUST be in router.go"** — чистое разделение ответственности в main.go.
8. **"NO duplicate response helpers"** — один `httputil` пакет вместо дублирования в middleware.
