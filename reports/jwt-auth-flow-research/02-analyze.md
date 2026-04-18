# JWT Auth Flow: Анализ

## Полный путь запроса

```
HTTP Request
  │
  ├─ Header: Authorization: Bearer <jwt>
  ├─ Header: X-Org-ID: <uuid>
  │
  ▼
middleware/auth.go — ValidateJWT
  │ 1. Извлекает Bearer token из Authorization header
  │ 2. Парсит JWT через golang-jwt/jwt/v5 (HMAC secret, не JWKS!)
  │ 3. Читает claims.Orgs []OrgClaim из токена
  │ 4. Парсит X-Org-ID header как UUID
  │ 5. Ищет X-Org-ID в claims.Orgs → получает role
  │ 6. Если не найден → 403 FORBIDDEN
  │ 7. Кладёт в context: org_id, role, org_ids[]
  │
  ▼
middleware/auth.go — RequireRole("viewer", "editor", ...)
  │ Читает role из context
  │ Проверяет что role есть в списке разрешённых
  │ Если нет → 403 FORBIDDEN
  │
  ▼
handler/organization.go
  │ Извлекает org_ids[] или org_id из context
  │ Для organizations: проверяет membership через slices.Contains
  │ Для sub-resources: передаёт org_id в service
  │
  ▼
service/organization.go
  │ Бизнес-логика + OTel span + slog
  │ Прокидывает org_id / org_ids в repository
  │
  ▼
repository/organization.go
  │ Обёртка над sqlc-generated Queries
  │ Конвертация uuid.UUID → pgtype.UUID
  │ pgx.ErrNoRows → ErrNotFound
  │
  ▼
db/query.sql.go (sqlc-generated)
  │ Organizations: SELECT ... WHERE id = $1
  │ List: SELECT ... WHERE id = ANY($1::uuid[])
  │ Sub-resources: SELECT ... WHERE org_id = $1 AND id = $2
```

## Два паттерна доступа

### 1. Organizations (top-level, нет org_id колонки)
- Доступ через JWT membership: `orgIDs` из `claims.Orgs`
- `GetByID`: handler проверяет `slices.Contains(orgIDs, id)`
- `List`: передаёт `orgIDs` → SQL `WHERE id = ANY($1::uuid[])`
- org_id НЕ доходит до SQL — фильтрация на уровне handler

### 2. Sub-resources (campaigns, api_keys, etc.)
- Всегда `WHERE org_id = $1 AND id = $2`
- org_id берётся ТОЛЬКО из context (из JWT), никогда из request body
- Фильтрация на уровне SQL

## Ключевые проблемы безопасности

| # | Severity | Проблема | Файл |
|---|----------|----------|------|
| 1 | CRITICAL | HMAC secret вместо JWKS | middleware/auth.go:51 |
| 2 | CRITICAL | Нет проверки signing algorithm (alg confusion) | middleware/auth.go:51-56 |
| 3 | CRITICAL | Viewer может POST (создавать организации) | router/router.go:31-36 |
| 4 | HIGH | Нет проверки issuer/audience | middleware/auth.go:51-56 |
| 5 | HIGH | exp не обязателен — токен может жить вечно | middleware/auth.go:51-56 |
| 6 | HIGH | Нет ограничения на размер org_ids массива | repository/organization.go |

## Что сделано правильно

- X-Org-ID валидируется против JWT claims (нельзя подставить чужой org)
- org_id никогда не берётся из request body
- sqlc исключает SQL injection
- Sentinel errors вместо raw DB errors
- Единый response helper для middleware и handlers
