-- name: CreateDeployment :one
INSERT INTO deployments (app_id, version, image, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetDeploymentByID :one
SELECT * FROM deployments WHERE id = $1;

-- name: ListDeploymentsByApp :many
SELECT * FROM deployments
WHERE app_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetLatestDeployment :one
SELECT * FROM deployments
WHERE app_id = $1
ORDER BY version DESC
LIMIT 1;

-- name: UpdateDeploymentStatus :one
UPDATE deployments
SET status = $2, message = $3, error = $4
WHERE id = $1
RETURNING *;

-- name: UpdateDeploymentStarted :one
UPDATE deployments
SET status = 'building', started_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDeploymentReady :one
UPDATE deployments
SET status = 'running', ready_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDeploymentFailed :one
UPDATE deployments
SET status = 'failed', error = $2
WHERE id = $1
RETURNING *;

-- name: DeleteDeployment :exec
DELETE FROM deployments WHERE id = $1;

-- name: CountDeploymentsByApp :one
SELECT COUNT(*) FROM deployments WHERE app_id = $1;
