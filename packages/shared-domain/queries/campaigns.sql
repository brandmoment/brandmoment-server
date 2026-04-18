-- name: GetCampaignByID :one
SELECT id, org_id, name, status, targeting, budget_cents, currency,
       start_date, end_date, created_at, updated_at
FROM campaigns
WHERE org_id = @org_id AND id = @id;

-- name: ListCampaignsByOrg :many
SELECT id, org_id, name, status, targeting, budget_cents, currency,
       start_date, end_date, created_at, updated_at
FROM campaigns
WHERE org_id = @org_id
  AND (sqlc.narg('status_filter')::TEXT IS NULL OR status = sqlc.narg('status_filter'))
ORDER BY created_at DESC
LIMIT @limit_val OFFSET @offset_val;

-- name: CountCampaignsByOrg :one
SELECT COUNT(*) FROM campaigns
WHERE org_id = @org_id
  AND (sqlc.narg('status_filter')::TEXT IS NULL OR status = sqlc.narg('status_filter'));

-- name: InsertCampaign :one
INSERT INTO campaigns (id, org_id, name, status, targeting, budget_cents, currency,
                       start_date, end_date, created_at, updated_at)
VALUES (@id, @org_id, @name, @status, @targeting, @budget_cents, @currency,
        @start_date, @end_date, @created_at, @updated_at)
RETURNING *;

-- name: UpdateCampaign :one
UPDATE campaigns
SET name = @name, targeting = @targeting, budget_cents = @budget_cents,
    currency = @currency, start_date = @start_date, end_date = @end_date,
    updated_at = @updated_at
WHERE org_id = @org_id AND id = @id
RETURNING *;

-- name: UpdateCampaignStatus :one
UPDATE campaigns
SET status = @status, updated_at = @updated_at
WHERE org_id = @org_id AND id = @id
RETURNING *;
