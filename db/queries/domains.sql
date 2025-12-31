-- name: CreateDomain :one
INSERT INTO domains (app_id, domain)
VALUES ($1, $2)
RETURNING *;

-- name: GetDomainByID :one
SELECT * FROM domains WHERE id = $1;

-- name: GetDomainByName :one
SELECT * FROM domains WHERE domain = $1;

-- name: ListDomainsByApp :many
SELECT * FROM domains
WHERE app_id = $1
ORDER BY created_at DESC;

-- name: UpdateDomainVerified :one
UPDATE domains
SET verified = TRUE, verified_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDomainSSLStatus :one
UPDATE domains
SET ssl_status = $2
WHERE id = $1
RETURNING *;

-- name: DeleteDomain :exec
DELETE FROM domains WHERE id = $1;

-- name: CountDomainsByApp :one
SELECT COUNT(*) FROM domains WHERE app_id = $1;
