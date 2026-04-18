-- name: GetMembershipByUserAndOrg :one
SELECT id, org_id, user_id, role, created_at
FROM org_memberships
WHERE user_id = @user_id AND org_id = @org_id;

-- name: ListMembershipsByUser :many
SELECT id, org_id, user_id, role, created_at
FROM org_memberships
WHERE user_id = @user_id
ORDER BY created_at ASC;

-- name: InsertMembership :one
INSERT INTO org_memberships (id, org_id, user_id, role, created_at)
VALUES (@id, @org_id, @user_id, @role, @created_at)
RETURNING *;
