-- name: CreateUser :one
INSERT INTO users (github_id, username, email, avatar_url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByGitHubID :one
SELECT * FROM users WHERE github_id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: UpdateUser :one
UPDATE users
SET username = $2, email = $3, avatar_url = $4
WHERE id = $1
RETURNING *;

-- name: UpdateUserPlan :one
UPDATE users
SET plan = $2, stripe_customer_id = $3
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: UpdateUserEmail :exec
UPDATE users SET email = $2 WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
