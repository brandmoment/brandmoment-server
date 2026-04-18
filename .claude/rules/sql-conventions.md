---
description: SQL migration conventions, sqlc configuration, and database naming rules
globs: "**/*.sql"
---

# SQL & Database Rules

## Migrations

Location: `infra/migrations/`. Tool: golang-migrate. Naming: `NNNNNN_<description>.up.sql` + `.down.sql`.

Rules:
1. Always both `.up.sql` and `.down.sql`
2. Use `IF NOT EXISTS` / `IF EXISTS` for idempotency
3. `TIMESTAMPTZ` (not TIMESTAMP) for all time columns
4. `gen_random_uuid()` for UUID defaults
5. Index columns used in WHERE/JOIN (especially `org_id`, `slug`)
6. Sub-resource tables MUST have `org_id UUID NOT NULL REFERENCES organizations(id)`
7. Sequential numbering: `000001`, `000002`, etc.
8. Enums: TEXT with CHECK constraint (not Postgres ENUM type)

## sqlc

Mandatory for all DB queries. No raw SQL in Go code.

Config: `packages/shared-domain/sqlc.yaml` (engine: postgresql, sql_package: pgx/v5, package: db).
Queries: `packages/shared-domain/queries/<entity>.sql`.

Naming: `Get<Entity>ByID` (:one), `List<Entity>sByOrg` (:many), `Insert<Entity>` (:one RETURNING), `Update<Entity>` (:one RETURNING), `Delete<Entity>` (:exec).

Use `@param_name` syntax for named parameters. After writing queries — run `sqlc generate`, then wrap in repository.
