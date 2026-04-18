-- name: GetPublisherAppByID :one
SELECT id, org_id, name, platform, bundle_id, is_active, created_at, updated_at
FROM publisher_apps
WHERE org_id = @org_id AND id = @id;

-- name: GetPublisherAppByBundleID :one
SELECT id, org_id, name, platform, bundle_id, is_active, created_at, updated_at
FROM publisher_apps
WHERE org_id = @org_id AND bundle_id = @bundle_id;

-- name: ListPublisherAppsByOrg :many
SELECT id, org_id, name, platform, bundle_id, is_active, created_at, updated_at
FROM publisher_apps
WHERE org_id = @org_id
ORDER BY created_at DESC
LIMIT @limit_val OFFSET @offset_val;

-- name: CountPublisherAppsByOrg :one
SELECT COUNT(*) FROM publisher_apps
WHERE org_id = @org_id;

-- name: InsertPublisherApp :one
INSERT INTO publisher_apps (id, org_id, name, platform, bundle_id, is_active, created_at, updated_at)
VALUES (@id, @org_id, @name, @platform, @bundle_id, @is_active, @created_at, @updated_at)
RETURNING *;

-- name: UpdatePublisherApp :one
UPDATE publisher_apps
SET name       = @name,
    is_active  = @is_active,
    updated_at = @updated_at
WHERE org_id = @org_id AND id = @id
RETURNING *;
