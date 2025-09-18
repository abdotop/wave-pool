-- name: GetCheckoutSession :one
SELECT *
FROM checkout_sessions
WHERE id = ?;

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
	wave_launch_url,
	when_created,
	when_expires,
	when_completed,
	when_refunded,
	last_payment_error_code,
	last_payment_error_message
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);
