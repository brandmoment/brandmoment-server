-- name: GetAPIKeyByID :one
SELECT id, org_id, app_id, name, key_hash, key_prefix, is_revoked, created_at, revoked_at
FROM api_keys
WHERE org_id = @org_id AND app_id = @app_id AND id = @id;

-- name: GetAPIKeyByHash :one
SELECT id, org_id, app_id, name, key_hash, key_prefix, is_revoked, created_at, revoked_at
FROM api_keys
WHERE key_hash = @key_hash;

-- name: ListAPIKeysByApp :many
SELECT id, org_id, app_id, name, key_hash, key_prefix, is_revoked, created_at, revoked_at
FROM api_keys
WHERE org_id = @org_id AND app_id = @app_id AND (NOT @active_only OR is_revoked = false)
ORDER BY created_at DESC;

-- name: InsertAPIKey :one
INSERT INTO api_keys (id, org_id, app_id, name, key_hash, key_prefix, is_revoked, created_at)
VALUES (@id, @org_id, @app_id, @name, @key_hash, @key_prefix, false, @created_at)
RETURNING *;

-- name: RevokeAPIKey :one
UPDATE api_keys
SET is_revoked = true,
    revoked_at = @revoked_at
WHERE org_id = @org_id AND app_id = @app_id AND id = @id
RETURNING *;
