-- name: GetPublisherRuleByID :one
SELECT id, org_id, app_id, type, config, is_active, created_at, updated_at
FROM publisher_rules
WHERE org_id = @org_id AND app_id = @app_id AND id = @id;

-- name: ListPublisherRulesByApp :many
SELECT id, org_id, app_id, type, config, is_active, created_at, updated_at
FROM publisher_rules
WHERE org_id = @org_id AND app_id = @app_id
ORDER BY created_at DESC
LIMIT @limit_val OFFSET @offset_val;

-- name: CountPublisherRulesByApp :one
SELECT COUNT(*) FROM publisher_rules
WHERE org_id = @org_id AND app_id = @app_id;

-- name: InsertPublisherRule :one
INSERT INTO publisher_rules (id, org_id, app_id, type, config, is_active, created_at, updated_at)
VALUES (@id, @org_id, @app_id, @type, @config, @is_active, @created_at, @updated_at)
RETURNING *;

-- name: UpdatePublisherRule :one
UPDATE publisher_rules
SET config     = @config,
    is_active  = @is_active,
    updated_at = @updated_at
WHERE org_id = @org_id AND app_id = @app_id AND id = @id
RETURNING *;

-- name: DeletePublisherRule :exec
DELETE FROM publisher_rules
WHERE org_id = @org_id AND app_id = @app_id AND id = @id;
