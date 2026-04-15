-- name: GetOrganizationByID :one
SELECT id, type, name, slug, created_at, updated_at
FROM organizations
WHERE id = @id;

-- name: ListOrganizationsByIDs :many
SELECT id, type, name, slug, created_at, updated_at
FROM organizations
WHERE id = ANY(@ids::uuid[])
ORDER BY created_at DESC;

-- name: InsertOrganization :one
INSERT INTO organizations (id, type, name, slug, created_at, updated_at)
VALUES (@id, @type, @name, @slug, @created_at, @updated_at)
RETURNING *;
