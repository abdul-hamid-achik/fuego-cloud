-- name: CreateApp :one
INSERT INTO apps (user_id, name, region, size)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAppByID :one
SELECT * FROM apps WHERE id = $1;

-- name: GetAppByName :one
SELECT * FROM apps
WHERE user_id = $1 AND name = $2;

-- name: ListAppsByUser :many
SELECT * FROM apps
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateApp :one
UPDATE apps
SET name = $2, region = $3, size = $4
WHERE id = $1
RETURNING *;

-- name: UpdateAppStatus :one
UPDATE apps
SET status = $2, current_deployment_id = $3
WHERE id = $1
RETURNING *;

-- name: IncrementDeploymentCount :one
UPDATE apps
SET deployment_count = deployment_count + 1
WHERE id = $1
RETURNING *;

-- name: UpdateAppEnvVars :one
UPDATE apps
SET env_vars_encrypted = $2
WHERE id = $1
RETURNING *;

-- name: DeleteApp :exec
DELETE FROM apps WHERE id = $1;

-- name: CountAppsByUser :one
SELECT COUNT(*) FROM apps WHERE user_id = $1;
