-- name: CreateCheckoutSession :one
INSERT INTO checkout_sessions (
    id,
    business_id,
    amount,
    currency,
    client_reference,
    aggregated_merchant_id,
    status,
    error_url,
    success_url,
    restrict_payer_mobile,
    wave_launch_url,
    transaction_id,
    payment_status,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: GetCheckoutSession :one
SELECT * FROM checkout_sessions
WHERE id = $1 AND business_id = $2;

-- name: GetCheckoutSessionByID :one
SELECT * FROM checkout_sessions
WHERE id = $1;

-- name: GetCheckoutSessionByTxID :one
SELECT * FROM checkout_sessions
WHERE transaction_id = $1 AND business_id = $2
LIMIT 1;

-- name: SearchCheckoutSessions :many
SELECT * FROM checkout_sessions
WHERE business_id = $1
  AND client_reference = $2
ORDER BY when_created DESC;

-- name: UpdateCheckoutPaymentStatus :exec
UPDATE checkout_sessions
SET payment_status = $2
WHERE id = $1 AND business_id = $3;

-- name: ExpireCheckoutSession :exec
UPDATE checkout_sessions
SET    status = $2,
       when_completed = $3
WHERE  id = $1
  AND  status = 'open';

-- name: SucceedCheckoutSession :one
UPDATE checkout_sessions
SET    status = 'complete',
       payment_status = 'succeeded',
       when_completed = now()
WHERE  id = $1
RETURNING *;

-- name: FailCheckoutSession :one
UPDATE checkout_sessions
SET    payment_status = 'cancelled',
       last_payment_error = $2
WHERE  id = $1
RETURNING *;

-- name: CreatePayment :one
INSERT INTO payments (
    id,
    session_id,
    amount,
    currency,
    status,
    failure_reason
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetAPIKeyByPrefixAndSecret :one
SELECT k.*, b.id as business_id_alias, b.name as business_name
FROM api_keys k
JOIN business b ON k.business_id = b.id
WHERE k.prefix = $1 AND k.key_hash = $2;
-- wave-pool_pro_q955KEAC71StVaoYzkf4apSx9sqolaOONVDVEdCPI5s