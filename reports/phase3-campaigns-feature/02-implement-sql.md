# SQL Implementation — Phase 3 Campaign/Creative

Agent: sql-builder
Stage: Implement (SQL)
Date: 2026-04-18

## Files Created

### Migrations

`infra/migrations/000008_create_campaigns.up.sql`
`infra/migrations/000008_create_campaigns.down.sql`
`infra/migrations/000009_create_creatives.up.sql`
`infra/migrations/000009_create_creatives.down.sql`

### sqlc Query Files

`packages/shared-domain/queries/campaigns.sql`
`packages/shared-domain/queries/creatives.sql`

### Generated Go Code (by sqlc)

`packages/shared-domain/db/campaigns.sql.go` — 6.2K
`packages/shared-domain/db/creatives.sql.go` — 4.0K
`packages/shared-domain/db/models.go` — updated with Campaign and Creative structs

---

## sqlc generate Result

Ran `sqlc generate` in `packages/shared-domain/`. Exited with code 0, no errors or warnings.

---

## Generated Type Map (critical for go-builder)

### Campaign model fields (from generated `Campaign` struct in `models.go`)

| Column         | Go type             | Notes                                      |
|----------------|---------------------|--------------------------------------------|
| id             | pgtype.UUID         |                                            |
| org_id         | pgtype.UUID         |                                            |
| name           | string              |                                            |
| status         | string              |                                            |
| targeting      | []byte              | JSONB — marshal/unmarshal in repository    |
| budget_cents   | pgtype.Int8         | Nullable BIGINT                            |
| currency       | string              |                                            |
| start_date     | pgtype.Date         | Nullable DATE                              |
| end_date       | pgtype.Date         | Nullable DATE                              |
| created_at     | pgtype.Timestamptz  |                                            |
| updated_at     | pgtype.Timestamptz  |                                            |

### Creative model fields

| Column          | Go type            | Notes                    |
|-----------------|--------------------|--------------------------|
| id              | pgtype.UUID        |                          |
| org_id          | pgtype.UUID        |                          |
| campaign_id     | pgtype.UUID        |                          |
| name            | string             |                          |
| type            | string             |                          |
| file_url        | string             |                          |
| file_size_bytes | pgtype.Int8        | Nullable BIGINT          |
| preview_url     | pgtype.Text        | Nullable TEXT            |
| is_active       | bool               |                          |
| created_at      | pgtype.Timestamptz |                          |
| updated_at      | pgtype.Timestamptz |                          |

---

## Known Issues for go-builder

### 1. status_filter — RESOLVED

Initially generated as plain `string`. Fixed by using `sqlc.narg('status_filter')` in both `ListCampaignsByOrg` and `CountCampaignsByOrg`. After re-running `sqlc generate`, `StatusFilter` is now `pgtype.Text`.

Repository usage:
- No filter: `StatusFilter: pgtype.Text{Valid: false}`
- Filter by status: `StatusFilter: pgtype.Text{String: "active", Valid: true}`

### 2. targeting is []byte — needs json.Marshal/Unmarshal

The repository `toCampaign(row db.Campaign)` helper must `json.Unmarshal(row.Targeting, &model.CampaignTargeting{})` on read. The `fromCampaign` helper must `json.Marshal(c.Targeting)` before passing to sqlc params.

### 3. pgtype.Date ↔ *time.Time conversion

`start_date` and `end_date` are `pgtype.Date` in generated code. The repository helpers must convert:
- Read: `if row.StartDate.Valid { t := row.StartDate.Time; c.StartDate = &t }`
- Write: `if c.StartDate != nil { pgtype.Date{Time: *c.StartDate, Valid: true} } else { pgtype.Date{} }`

### 4. pgtype.Int8 ↔ *int64 conversion

Same pattern as above for `budget_cents` and `file_size_bytes`.

### 5. pgtype.Text ↔ *string conversion

`preview_url` is `pgtype.Text`. Same nullable conversion pattern.

---

## Indexes Summary

### campaigns
- `idx_campaigns_org_id` — covers all `WHERE org_id = $1` queries
- `idx_campaigns_org_id_status` — covers `WHERE org_id = $1 AND status = $2` (list with filter)
- `idx_campaigns_org_id_created` — covers `ORDER BY created_at DESC` scans per org

### creatives
- `idx_creatives_campaign_id` — covers `WHERE campaign_id = $1` joins and lookups
- `idx_creatives_org_id` — covers `WHERE org_id = $1` cross-campaign lookups
- `idx_creatives_campaign_id_active` — covers `WHERE campaign_id = $1 AND is_active = true` future queries

---

## Queries Implemented

### campaigns.sql

| Query name            | Type  | Description                                      |
|-----------------------|-------|--------------------------------------------------|
| GetCampaignByID       | :one  | Fetch single campaign by org_id + id             |
| ListCampaignsByOrg    | :many | Paginated list with optional status filter       |
| CountCampaignsByOrg   | :one  | Count for pagination total (same status filter)  |
| InsertCampaign        | :one  | Create campaign, RETURNING *                     |
| UpdateCampaign        | :one  | Update name/targeting/budget/dates, RETURNING *  |
| UpdateCampaignStatus  | :one  | Status-only update, RETURNING *                  |

### creatives.sql

| Query name               | Type  | Description                                   |
|--------------------------|-------|-----------------------------------------------|
| GetCreativeByID          | :one  | Fetch single creative by org_id+campaign_id+id|
| ListCreativesByCampaign  | :many | List all creatives for campaign (hard cap 50) |
| CountCreativesByCampaign | :one  | Count creatives for campaign                  |
| InsertCreative           | :one  | Create creative, RETURNING *                  |

---

## Multi-Tenancy Verification

- Every campaign query includes `WHERE org_id = @org_id` — correct.
- Every creative query includes `WHERE org_id = @org_id AND campaign_id = @campaign_id` — correct.
- No query allows access to campaigns or creatives across org boundaries.

---

## Next Step for go-builder

The `sqlc.narg` fix for `status_filter` was applied to `campaigns.sql` and `sqlc generate` was re-run successfully. `StatusFilter` is now `pgtype.Text` in `ListCampaignsByOrgParams` and `CountCampaignsByOrgParams`. No further sqlc changes needed before implementing repositories.

Files to implement next (in order per spec):
1. `services/api-dashboard/internal/model/campaign.go`
2. `services/api-dashboard/internal/model/creative.go`
3. `services/api-dashboard/internal/repository/campaign.go`
4. `services/api-dashboard/internal/repository/creative.go`
5. `services/api-dashboard/internal/service/campaign.go`
6. `services/api-dashboard/internal/service/creative.go`
7. `services/api-dashboard/internal/service/campaign_test.go`
8. `services/api-dashboard/internal/service/creative_test.go`
9. `services/api-dashboard/internal/handler/campaign.go`
10. `services/api-dashboard/internal/handler/creative.go`
11. `services/api-dashboard/internal/router/router.go` (add routes)
12. `services/api-dashboard/cmd/server/main.go` (DI wiring)
