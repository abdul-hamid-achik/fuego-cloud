-- name: CreateOAuthState :one
INSERT INTO oauth_states (state, redirect_uri, cli_token_exchange, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetOAuthState :one
SELECT * FROM oauth_states WHERE state = $1;

-- name: DeleteOAuthState :exec
DELETE FROM oauth_states WHERE state = $1;

-- name: DeleteExpiredOAuthStates :exec
DELETE FROM oauth_states WHERE expires_at < NOW();
