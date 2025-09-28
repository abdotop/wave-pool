-- +goose Up
-- +goose StatementBegin
ALTER TABLE secrets ADD COLUMN webhook_url TEXT;
ALTER TABLE secrets ADD COLUMN webhook_security_strategy TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE secrets DROP COLUMN webhook_security_strategy;
ALTER TABLE secrets DROP COLUMN webhook_url;
-- +goose StatementEnd