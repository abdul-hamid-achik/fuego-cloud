-- name: CreateActivityLog :one
INSERT INTO activity_logs (user_id, app_id, action, details, ip_address)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListActivityLogsByApp :many
SELECT * FROM activity_logs
WHERE app_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListActivityLogsByUser :many
SELECT * FROM activity_logs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountActivityLogsByApp :one
SELECT COUNT(*) FROM activity_logs
WHERE app_id = $1;
