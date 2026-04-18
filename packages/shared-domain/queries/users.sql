-- name: GetUserByID :one
SELECT id, email, name, created_at
FROM users
WHERE id = @id;

-- name: GetUserByEmail :one
SELECT id, email, name, created_at
FROM users
WHERE email = @email;

-- name: UpsertUser :one
INSERT INTO users (id, email, name, created_at)
VALUES (@id, @email, @name, @created_at)
ON CONFLICT (email) DO UPDATE
  SET name = EXCLUDED.name
RETURNING *;
