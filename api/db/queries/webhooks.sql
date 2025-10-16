-- name: CreateWebhook :one
INSERT INTO webhooks (
    id,
    business_id,
    url,
    signing_strategy,
    secret,
    events,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetWebhookByID :one
SELECT * FROM webhooks
WHERE id = $1 AND business_id = $2;

-- name: ListWebhooksByBusinessID :many
SELECT * FROM webhooks
WHERE business_id = $1
ORDER BY created_at DESC;

-- name: UpdateWebhook :one
UPDATE webhooks
SET
    url = $2,
    signing_strategy = $3,
    events = $4,
    status = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteWebhook :exec
DELETE FROM webhooks
WHERE id = $1;