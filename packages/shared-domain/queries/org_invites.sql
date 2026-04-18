-- name: InsertOrgInvite :one
INSERT INTO org_invites (id, org_id, email, role, token, expires_at, created_at)
VALUES (@id, @org_id, @email, @role, @token, @expires_at, @created_at)
RETURNING *;

-- name: GetOrgInviteByToken :one
SELECT id, org_id, email, role, token, expires_at, accepted_at, created_at
FROM org_invites
WHERE token = @token;
