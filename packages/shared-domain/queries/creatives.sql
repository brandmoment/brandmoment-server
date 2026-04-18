-- name: GetCreativeByID :one
SELECT id, org_id, campaign_id, name, type, file_url, file_size_bytes,
       preview_url, is_active, created_at, updated_at
FROM creatives
WHERE org_id = @org_id AND campaign_id = @campaign_id AND id = @id;

-- name: ListCreativesByCampaign :many
SELECT id, org_id, campaign_id, name, type, file_url, file_size_bytes,
       preview_url, is_active, created_at, updated_at
FROM creatives
WHERE org_id = @org_id AND campaign_id = @campaign_id
ORDER BY created_at ASC
LIMIT 50;

-- name: CountCreativesByCampaign :one
SELECT COUNT(*) FROM creatives
WHERE org_id = @org_id AND campaign_id = @campaign_id;

-- name: InsertCreative :one
INSERT INTO creatives (id, org_id, campaign_id, name, type, file_url,
                       file_size_bytes, preview_url, is_active, created_at, updated_at)
VALUES (@id, @org_id, @campaign_id, @name, @type, @file_url,
        @file_size_bytes, @preview_url, @is_active, @created_at, @updated_at)
RETURNING *;
