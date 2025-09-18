-- +goose Up
-- +goose StatementBegin
ALTER TABLE secrets ADD COLUMN security_strategy TEXT CHECK (security_strategy IN ('SHARED_SECRET', 'SIGNING_SECRET'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE secrets DROP COLUMN security_strategy;
-- +goose StatementEnd
