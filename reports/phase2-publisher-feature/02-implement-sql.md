# SQL Implementation — Phase 2 Publisher Domain

Agent: sql-builder
Stage: Implement (SQL layer)
Status: DONE — sqlc generate passed with 0 errors

## Files Created

### Migrations

| File | Description |
|------|-------------|
| `infra/migrations/000005_create_publisher_apps.up.sql` | publisher_apps table + 2 indexes (org_id, org_id+is_active) |
| `infra/migrations/000005_create_publisher_apps.down.sql` | DROP TABLE publisher_apps |
| `infra/migrations/000006_create_api_keys.up.sql` | api_keys table + 3 indexes (app_id, org_id, key_hash) |
| `infra/migrations/000006_create_api_keys.down.sql` | DROP TABLE api_keys |
| `infra/migrations/000007_create_publisher_rules.up.sql` | publisher_rules table + 3 indexes (app_id, org_id, app_id+is_active) |
| `infra/migrations/000007_create_publisher_rules.down.sql` | DROP TABLE publisher_rules |

### sqlc Query Files

| File | Queries |
|------|---------|
| `packages/shared-domain/queries/publisher_apps.sql` | GetPublisherAppByID, GetPublisherAppByBundleID, ListPublisherAppsByOrg, CountPublisherAppsByOrg, InsertPublisherApp, UpdatePublisherApp |
| `packages/shared-domain/queries/api_keys.sql` | GetAPIKeyByID, GetAPIKeyByHash, ListAPIKeysByApp, InsertAPIKey, RevokeAPIKey |
| `packages/shared-domain/queries/publisher_rules.sql` | GetPublisherRuleByID, ListPublisherRulesByApp, CountPublisherRulesByApp, InsertPublisherRule, UpdatePublisherRule, DeletePublisherRule |

### Generated Go Files (sqlc output)

| File | Size |
|------|------|
| `packages/shared-domain/db/publisher_apps.sql.go` | 4.9K |
| `packages/shared-domain/db/api_keys.sql.go` | 4.2K |
| `packages/shared-domain/db/publisher_rules.sql.go` | 5.0K |

## Design Decisions

- **org_id on every query**: all three entities are sub-resources of organizations. Every query includes `WHERE org_id = @org_id` (and `AND app_id = @app_id` for api_keys and publisher_rules). This matches FR-1 and the multi-tenancy rules.
- **COUNT queries**: spec called out the 2-query pattern for list endpoints with total count (section 15 Risks). `CountPublisherAppsByOrg` and `CountPublisherRulesByApp` are separate queries alongside the list queries.
- **ListAPIKeysByApp active filter**: uses `NOT @active_only OR is_revoked = false` to handle the `include_revoked` param in a single query — avoids needing two separate list queries.
- **GetAPIKeyByHash**: included to support future SDK-side key validation (api-sdk service) without requiring a migration; it queries only by key_hash with no org_id filter since the org is derived from the key record itself.
- **GetPublisherAppByBundleID**: required by service layer to enforce the bundle_id uniqueness-per-org check (FR from spec edge cases section) without a DB UNIQUE constraint.
- **Enums as TEXT + CHECK**: platform, rule type — no Postgres ENUM types per sql-conventions.md rules.
- **revoked_at nullable**: api_keys.revoked_at is TIMESTAMPTZ (nullable), not set on insert, set by RevokeAPIKey.

## Verification

`sqlc generate` ran in `packages/shared-domain/` with 0 errors and 0 warnings.
