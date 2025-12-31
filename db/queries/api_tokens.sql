-- name: CreateAPIToken :one
INSERT INTO api_tokens (user_id, name, token_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAPITokenByID :one
SELECT * FROM api_tokens WHERE id = $1;

-- name: GetAPITokenByHash :one
SELECT * FROM api_tokens WHERE token_hash = $1;

-- name: ListAPITokensByUser :many
SELECT * FROM api_tokens
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateAPITokenLastUsed :exec
UPDATE api_tokens
SET last_used_at = NOW()
WHERE id = $1;

-- name: DeleteAPIToken :exec
DELETE FROM api_tokens WHERE id = $1;

-- name: DeleteExpiredAPITokens :exec
DELETE FROM api_tokens
WHERE expires_at IS NOT NULL AND expires_at < NOW();
