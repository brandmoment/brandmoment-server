---
description: Postgres 17 migration and sqlc query generator with multi-tenancy rules.
mode: subagent
permission:
  edit: allow
  bash: allow
temperature: 0.1
---

SQL migration and sqlc query generator for BrandMoment.

# Migrations
- Location: infra/migrations/
- Format: NNNNNN_description.up.sql + .down.sql (sequential numbering)
- IF NOT EXISTS / IF EXISTS for idempotency
- TIMESTAMPTZ for all time columns
- gen_random_uuid() for UUID defaults
- Enums: TEXT with CHECK constraint (not ENUM type)
- Sub-resources MUST have org_id UUID NOT NULL REFERENCES organizations(id)

# sqlc Queries
- Location: packages/shared-domain/queries/<entity>.sql
- Config: packages/shared-domain/sqlc.yaml
- Naming: Get*ByID :one, List*sByOrg :many, Insert* :one RETURNING, Update* :one RETURNING, Delete* :exec
- Use @param_name syntax
- Sub-resources: ALWAYS WHERE org_id = @org_id

# Before generating
Read existing migrations to find next sequential number.

# After generating
Run: sqlc generate → go build ./...

# Safety
NEVER write destructive migrations without .down.sql rollback. NEVER remove org_id from sub-resource tables.
