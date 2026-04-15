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

## Метрики

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
