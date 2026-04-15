---
description: SQL migration conventions, sqlc configuration, and database naming rules
globs: "**/*.sql"
---

# SQL & Database Rules

## Migration Files

Location: `infra/migrations/`
Tool: golang-migrate
Naming: `NNNNNN_<description>.up.sql` + `NNNNNN_<description>.down.sql`

```sql
-- 000001_create_organizations.up.sql
CREATE TABLE IF NOT EXISTS organizations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type       TEXT NOT NULL CHECK (type IN ('admin', 'publisher', 'brand')),
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_organizations_slug ON organizations (slug);

-- 000001_create_organizations.down.sql
DROP TABLE IF EXISTS organizations;
```

### Migration Rules

1. Always include both `.up.sql` and `.down.sql`
2. Use `IF NOT EXISTS` / `IF EXISTS` for idempotency
3. Use `TIMESTAMPTZ` (not `TIMESTAMP`) for all time columns
4. Use `gen_random_uuid()` for UUID defaults
5. Add indexes for columns used in WHERE/JOIN (especially `org_id`, `slug`)
6. Sub-resource tables MUST have `org_id UUID NOT NULL REFERENCES organizations(id)`
7. Sequential numbering: `000001`, `000002`, etc.

## Table Naming

- Tables: `snake_case`, plural (`organizations`, `campaigns`, `api_keys`)
- PKs: `id UUID`
- FKs: `{entity}_id` (`org_id`, `campaign_id`)
- Timestamps: `created_at`, `updated_at` (TIMESTAMPTZ)
- Booleans: `is_active`, `is_revoked`
- Enums: TEXT with CHECK constraint (not Postgres ENUM type)
- JSONB columns: `targeting`, `config`, `metadata`

## sqlc Configuration

sqlc is MANDATORY for all database queries. No raw SQL in Go code.

Config file: `packages/shared-domain/sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "../../infra/migrations/"
    gen:
      go:
        package: "db"
        out: "db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
```

### sqlc Query Files

Location: `packages/shared-domain/queries/<entity>.sql`

```sql
-- name: GetCampaignByID :one
SELECT id, org_id, name, status, created_at, updated_at
FROM campaigns
WHERE org_id = @org_id AND id = @id;

-- name: ListCampaignsByOrg :many
SELECT id, org_id, name, status, created_at, updated_at
FROM campaigns
WHERE org_id = @org_id
ORDER BY created_at DESC
LIMIT @limit_val OFFSET @offset_val;

-- name: InsertCampaign :one
INSERT INTO campaigns (id, org_id, name, status, created_at, updated_at)
VALUES (@id, @org_id, @name, @status, @created_at, @updated_at)
RETURNING *;
```

### Query Naming Convention

- `Get<Entity>ByID` — single row by PK (`:one`)
- `List<Entity>sByOrg` — multiple rows filtered by org_id (`:many`)
- `Insert<Entity>` — create (`:one` with RETURNING)
- `Update<Entity>` — update (`:one` with RETURNING)
- `Delete<Entity>` — soft or hard delete (`:exec`)

After writing query files — run `sqlc generate` to create Go code, then wrap in repository.