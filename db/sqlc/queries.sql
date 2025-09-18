-- name: GetCheckoutSession :one
SELECT *
FROM checkout_sessions
WHERE id = ?;

-- name: GetCheckoutSessionByTransactionID :one
SELECT *
FROM checkout_sessions
WHERE transaction_id = ?;

-- name: GetCheckoutSessionsByClientReference :many
SELECT *
FROM checkout_sessions
WHERE client_reference = ?;

-- name: CreateCheckoutSession :exec
INSERT INTO checkout_sessions (
	id,
	amount,
	checkout_status,
	client_reference,
	currency,
	error_url,
	success_url,
	business_name,
	payment_status,
	transaction_id,
	aggregated_merchant_id,
	restrict_payer_mobile,
	enforce_payer_mobile,
	wave_launch_url,
	when_created,
	when_expires,
	when_completed,
	when_refunded,
	last_payment_error_code,
	last_payment_error_message
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateCheckoutSessionStatus :exec
UPDATE checkout_sessions 
SET checkout_status = ?, when_completed = ?
WHERE id = ?;

-- name: UpdateCheckoutSessionRefund :exec
UPDATE checkout_sessions 
SET when_refunded = ?
WHERE id = ?;

-- Users CRUD

-- name: CreateUser :exec
INSERT INTO users (
	id, phone_number, pin_hash
) VALUES (
	?, ?, ?
);

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone_number = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: UpdateUserPinHash :exec
UPDATE users SET pin_hash = ? WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- Secrets CRUD

-- name: CreateSecret :exec
INSERT INTO secrets (
	id, user_id, secret_hash, secret_type, permissions, display_hint, security_strategy
) VALUES (
	?, ?, ?, ?, ?, ?, ?
);

-- name: GetSecretByID :one
SELECT * FROM secrets WHERE id = ?;

-- name: GetSecretByHash :one
SELECT * FROM secrets WHERE secret_hash = ?;

-- name: ListSecretsByUser :many
SELECT * FROM secrets WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: RevokeSecret :exec
UPDATE secrets SET revoked_at = ? WHERE id = ?;

-- name: DeleteSecret :exec
DELETE FROM secrets WHERE id = ?;

-- Sessions CRUD

-- name: CreateSession :exec
INSERT INTO sessions (
	id, user_id, expires_at
) VALUES (
	?, ?, ?
);

-- name: GetSession :one
SELECT * FROM sessions WHERE id = ?;

-- name: ListSessionsByUser :many
SELECT * FROM sessions WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= ?;
