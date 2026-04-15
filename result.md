# Результат: CRUD Organizations для api-dashboard

## Что было сделано

### SQL миграция

| Файл | Описание |
|------|----------|
| `infra/migrations/000001_create_organizations.up.sql` | Таблица `organizations` (id UUID PK, type TEXT с CHECK constraint, name, slug UNIQUE, created_at, updated_at) + индексы на slug и type |
| `infra/migrations/000001_create_organizations.down.sql` | DROP TABLE organizations |

### Go-сервис (`services/api-dashboard/`)

| Слой | Файл | Что делает |
|------|------|------------|
| **model** | `internal/model/organization.go` | Структура `Organization`, enum `OrgType` (admin/publisher/brand) с валидацией, sentinel-ошибки `ErrNotFound`, `ErrInvalidInput` |
| **repository** | `internal/repository/organization.go` | Интерфейс `OrganizationRepository` + реализация на pgx. Insert, GetByID, ListByOrg — все запросы фильтруются по `org_id` |
| **service** | `internal/service/organization.go` | `OrganizationService` с OTel tracing (span на каждый метод, RecordError) + structured logging через slog |
| **middleware** | `internal/middleware/auth.go` | JWT HMAC-SHA256 валидация, извлечение `org_id` из payload, хелпер `OrgIDFromContext()` |
| **handler** | `internal/handler/organization.go` | HTTP-хендлеры: `POST /`, `GET /`, `GET /{id}` с парсингом query params (limit/offset) |
| **handler** | `internal/handler/response.go` | `respondJSON` / `respondError` / `handleServiceError` — стандартный envelope `{data, error}` |
| **handler** | `internal/handler/health.go` | `GET /healthz` — health check |
| **config** | `internal/config/config.go` | Конфигурация из env: `DATABASE_URL`, `JWT_SECRET`, `PORT`, `OTEL_EXPORTER_OTLP_ENDPOINT` |
| **main** | `cmd/server/main.go` | Entry point: загрузка конфига, подключение к Postgres (pgxpool), инициализация OTel TracerProvider, DI-wiring всех слоёв, chi router с middleware chain, graceful shutdown |
| **go.mod** | `go.mod` | Зависимости: chi, uuid, pgx, otelchi, otel SDK + gRPC exporter |

### Endpoints

| Метод | URL | Описание |
|-------|-----|----------|
| `GET` | `/healthz` | Health check (без auth) |
| `POST` | `/api/v1/organizations` | Создать организацию |
| `GET` | `/api/v1/organizations` | Список организаций (limit/offset) |
| `GET` | `/api/v1/organizations/:id` | Получить организацию по ID |

### Middleware chain

```
otelchi → RequestID → RealIP → Recoverer → [/api/v1: ValidateJWT] → handler
```

---

## Как строилась работа

### 1. Исследование кодовой базы

Первым шагом запустил **Agent (subagent_type: Explore)** для полного обзора монорепо. Агент проверил:
- Существующие файлы в `services/api-dashboard/` (только `go.mod` без кода)
- Наличие `infra/migrations/` (не существовало)
- `packages/shared-domain/` (только `go.mod`)
- `go.work` (workspace с двумя модулями)
- `Makefile`, `docker-compose.yml`, OTel config
- Все `go.mod` файлы в репо

Результат: репо полностью scaffolded (инфраструктура готова), но Go-код ещё не написан.

### 2. Планирование через задачи

Создал 7 задач через **TaskCreate** для трекинга прогресса:

1. SQL миграция
2. Модель + репозиторий
3. Сервис (OTel + slog)
4. Auth middleware
5. HTTP handlers + response helpers
6. Router, config, main.go
7. Проверка билда

Каждую задачу переводил в `in_progress` перед началом и `completed` после завершения через **TaskUpdate**.

### 3. Создание файлов

Использовал **Bash** для создания директорий (`mkdir -p`) и **Write** для создания каждого Go-файла. Файлы создавались в порядке зависимостей:
- Сначала model (нет зависимостей)
- Потом repository (зависит от model)
- Потом service (зависит от repository + model)
- Параллельно middleware (независим)
- Потом handler (зависит от service + middleware)
- В конце main.go (собирает всё)

Где возможно, независимые файлы создавались **параллельно** (несколько Write в одном сообщении).

### 4. Попытка сборки

Попытался запустить `go get` для скачивания зависимостей — обнаружил, что Go не установлен на машине. Проверил через `which go`, `brew list go`, поиск бинарника. Написал `go.mod` вручную с актуальными версиями зависимостей.

### 5. Ревью

Прочитал все созданные файлы через **Read** для финальной проверки:
- Корректность импортов (stdlib → third-party → internal)
- Кросс-ссылки между пакетами
- Соответствие конвенциям из CLAUDE.md

---

## Использованные инструменты (tools)

| Tool | Сколько раз | Для чего |
|------|-------------|----------|
| **Agent (Explore)** | 1 | Полное исследование структуры монорепо |
| **TaskCreate** | 7 | Создание задач для трекинга |
| **TaskUpdate** | 14 | Смена статусов задач (in_progress → completed) |
| **Write** | 12 | Создание всех Go-файлов и SQL миграций |
| **Read** | 5 | Чтение существующих файлов + финальный ревью созданного кода |
| **Bash** | 6 | mkdir, поиск Go binary, проверка файловой структуры |
| **ToolSearch** | 1 | Загрузка схем TaskCreate/TaskUpdate |
| **Edit** | 0 | Не потребовался (все файлы новые) |

## Использованные скиллы (skills)

Ни один skill (`/commit`, `/review`, и т.д.) не был вызван — задача не требовала их. Всё решалось базовыми инструментами.

---

## Что осталось сделать

```bash
# 1. Установить Go 1.23+
brew install go

# 2. Скачать зависимости
cd services/api-dashboard && go mod tidy

# 3. Проверить сборку
go build ./cmd/server/

# 4. Запустить инфру и применить миграцию
make infra-up
migrate -path infra/migrations -database "postgres://brandmoment:brandmoment@localhost:5432/brandmoment?sslmode=disable" up

# 5. Запустить сервер
DATABASE_URL="postgres://brandmoment:brandmoment@localhost:5432/brandmoment?sslmode=disable" \
JWT_SECRET="your-secret" \
go run ./cmd/server/
```
