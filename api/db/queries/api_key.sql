-- name: CreateAPIKey :one
INSERT INTO api_keys (id, business_id, prefix, key_hash, scopes, env)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetAPIKeyByID :one
SELECT *
FROM api_keys
WHERE id = $1;

-- name: ListAPIKeys :many
SELECT id, business_id, prefix, scopes, env, status, created_at
FROM api_keys
WHERE business_id = $1
ORDER BY created_at DESC;

-- name: RevokeAPIKey :exec
UPDATE api_keys
SET status = 'revoked'
WHERE id = $1 AND business_id = $2;