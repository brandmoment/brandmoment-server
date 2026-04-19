# SQL Implementation: UpdateCreative

## Status: Done

## Changes

### Query added
File: `packages/shared-domain/queries/creatives.sql`

Added `UpdateCreative :one` before `InsertCreative`. The query:
- Updates: `name`, `type`, `file_url`, `file_size_bytes`, `preview_url`, `is_active`, `updated_at = now()`
- Filters: `org_id = @org_id AND campaign_id = @campaign_id AND id = @id` (multi-tenancy enforced)
- Returns: `RETURNING *`
- Uses `@param_name` named parameter syntax throughout

### Generated Go code
File: `packages/shared-domain/db/creatives.sql.go`

`sqlc generate` ran clean (no errors). Generated:
- `const updateCreative` — SQL string at line 162
- `type UpdateCreativeParams struct` — params struct at line 175 (fields: ID, OrgID, CampaignID, Name, Type, FileURL, FileSizeBytes, PreviewURL, IsActive)
- `func (q *Queries) UpdateCreative(ctx context.Context, arg UpdateCreativeParams) (Creative, error)` — method at line 187

## Next steps
- Wrap `UpdateCreative` in the creatives repository (add method to repository interface + struct)
- Add service layer call
- Add HTTP handler (PUT /api/v1/orgs/{orgId}/campaigns/{campaignId}/creatives/{creativeId})
