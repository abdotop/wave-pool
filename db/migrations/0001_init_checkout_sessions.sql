-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS checkout_sessions (
    id TEXT PRIMARY KEY NOT NULL CHECK(length(id) <= 20),
    amount TEXT NOT NULL,
    checkout_status TEXT NOT NULL CHECK (checkout_status IN ('open','complete','expired')),
    client_reference TEXT,
    currency TEXT NOT NULL CHECK (length(currency) = 3 AND currency GLOB '[A-Z][A-Z][A-Z]'),
    error_url TEXT NOT NULL,
    success_url TEXT NOT NULL,
    business_name TEXT,
    payment_status TEXT NOT NULL CHECK (payment_status IN ('processing','cancelled','succeeded')),
    transaction_id TEXT UNIQUE,
    aggregated_merchant_id TEXT,
    restrict_payer_mobile TEXT,
    enforce_payer_mobile TEXT,
    wave_launch_url TEXT NOT NULL,
    when_created TEXT NOT NULL,
    when_expires TEXT NOT NULL,
    when_completed TEXT,
    when_refunded TEXT,
    last_payment_error_code TEXT,
    last_payment_error_message TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS checkout_sessions;
-- +goose StatementEnd
