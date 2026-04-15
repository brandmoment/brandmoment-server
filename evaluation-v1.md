# Оценка генерации #1: Organizations CRUD

## Общий вердикт

| Критерий | Оценка | Комментарий |
|----------|--------|-------------|
| Скомпилировалось? | **Нет** | Go не установлен, go.sum отсутствует, indirect deps не прописаны |
| Прошло линтер? | **Не проверено** | Но визуально: нет явных ошибок форматирования |
| Соответствует стилю? | **Частично (70%)** | Структура и naming верные, но есть концептуальные баги и отклонения от правил |

---

## Что сделано ПРАВИЛЬНО (соответствует CLAUDE.md)

### 1. Структура проекта
- `cmd/server/main.go` → `internal/{handler,service,repository,middleware,model,config}` — точно по шаблону

### 2. Naming conventions
- `OrganizationHandler`, `OrganizationService`, `OrganizationRepository` — PascalCase + суффикс
- `NewOrganizationService(deps)` — конструкторы
- `ErrNotFound`, `ErrInvalidInput` — sentinel errors
- Файлы: `organization.go`, `response.go` — snake_case

### 3. Import order
- Везде: stdlib → third-party → internal (с пустыми строками)

### 4. DI через конструкторы
- Никакого global state, никаких `init()` — зависимости передаются через `New*()`

### 5. Error handling
- Нет panic — все ошибки возвращаются через `error`
- `fmt.Errorf("insert organization: %w", err)` — wrapping с контекстом

### 6. Response format
- `{data, error}` envelope — точно по спеке

### 7. OTel tracing
- Span на каждый метод сервиса, `RecordError`, `defer span.End()`

### 8. Structured logging
- `slog.InfoContext`, `slog.DebugContext` с типизированными атрибутами

### 9. Middleware chain
- `otelchi → RequestID → RealIP → Recoverer → ValidateJWT` — по шаблону

### 10. Graceful shutdown
- Signal handling, context timeout — production-ready паттерн

### 11. SQL миграция
- `snake_case`, plural (`organizations`), UUID PK, `created_at`/`updated_at` — всё по конвенции

---

## Что сделано НЕПРАВИЛЬНО

### BUG 1: SQL-запросы в repository сломаны (критический)

**repository/organization.go:45** — `WHERE id = $1 AND id = $2`:
```go
// Сравнивает id с orgID И с id — бессмысленное условие
`SELECT ... FROM organizations WHERE id = $1 AND id = $2`, orgID, id
```

**repository/organization.go:64** — `WHERE id = $1`:
```go
// ListByOrg фильтрует по id = orgID — вернёт максимум одну строку
`SELECT ... FROM organizations WHERE id = $1 ORDER BY ...`, orgID
```

**Причина**: organizations — это top-level сущность. `org_id` фильтрация нужна для sub-resources (campaigns, apps), но не для самих организаций. У таблицы `organizations` нет колонки `org_id`. Ассистент слепо применил паттерн multi-tenancy к сущности, где он не применим.

### BUG 2: JWT claims не соответствуют спеке

**middleware/auth.go:34** — простая структура:
```go
type jwtClaims struct {
    OrgID string `json:"org_id"`
}
```

**Спека говорит**: JWT несёт массив org memberships с ролями:
```json
{"orgs": [{"org_id": "...", "role": "owner|admin|editor|viewer"}]}
```

Ассистент упростил до одного `org_id` — потеряна мульти-оргность и роли.

### BUG 3: Нет RBAC middleware

CLAUDE.md показывает:
```go
r.Use(auth.RequireRole("editor", "admin", "owner"))
```

В сгенерированном коде `middleware/rbac.go` **отсутствует**. Роутер не проверяет роли.

### ISSUE 4: Raw SQL вместо sqlc

CLAUDE.md указывает: **sqlc** для type-safe SQL generation. Repository использует raw `pool.Exec`/`pool.Query` с SQL строками. Нет:
- `sqlc.yaml` конфига
- `.sql` query файлов
- Сгенерированного кода

Это **нарушает** anti-pattern #3 ("NO raw SQL strings"), хотя формально SQL в repository layer (правильное место).

### ISSUE 5: Config без envconfig

CLAUDE.md: `config.go — Env-based config (envconfig)`. Сгенерирован plain `os.Getenv`. Не критично, но отклонение.

### ISSUE 6: Auth middleware дублирует response helpers

`writeAuthError()` в middleware/auth.go использует `map[string]any` вместо `Response`/`ErrorBody` из handler/response.go. Два разных формата ошибок.

### ISSUE 7: Router в main.go вместо отдельной функции

CLAUDE.md пример: `func NewRouter(h *Handlers, auth *middleware.Auth) http.Handler`. В main.go роутинг встроен прямо в `func main()`.

### ISSUE 8: Нет тестов

CLAUDE.md: `*_test.go`, `TestCreateCampaign`. Ни одного теста не сгенерировано.

---

## Что CLAUDE.md не покрыл (и ассистент не учёл)

| Пробел в правилах | Что произошло |
|---|---|
| Нет указания когда org_id filter НЕ применять | Ассистент слепо применил org_id ко всем запросам, сломав Organizations |
| Нет примера JWT claims из спеки | Ассистент придумал упрощённую структуру claims |
| Нет указания обязательно генерить sqlc конфиг | Ассистент написал raw SQL в repo |
| Нет правила про тесты | Ассистент не создал ни одного теста |
| Нет примера router extraction | Router оказался в main.go |
| Нет указания как middleware переиспользует response types | Получилось дублирование writeAuthError |
| Нет правила про валидацию request body (max sizes, slug format) | Ассистент валидирует только пустоту |

---

## Рекомендации для CLAUDE.md v2

1. **Уточнить multi-tenancy**: когда `org_id` нужен (sub-resources), когда нет (orgs сами), как работает доступ к организациям (через membership)
2. **Добавить JWT claims пример**: показать реальную структуру `{"orgs": [...]}`
3. **sqlc как обязательный**: добавить пример sqlc.yaml + query файлов
4. **Тесты обязательны**: минимум table-driven test на service layer
5. **Router extraction**: добавить правило выносить router в `internal/router/router.go`
6. **Middleware должен использовать общие response types**: запретить дублирование
7. **Добавить пример валидации**: slug format, name length limits
