-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS secrets (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    secret_hash TEXT NOT NULL UNIQUE,
    secret_type TEXT NOT NULL CHECK (secret_type IN ('API_KEY','WEBHOOK_SECRET')),
    permissions TEXT NOT NULL,
    display_hint TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_secrets_user_id ON secrets(user_id);
CREATE INDEX IF NOT EXISTS idx_secrets_type ON secrets(secret_type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_secrets_type;
DROP INDEX IF EXISTS idx_secrets_user_id;
DROP TABLE IF EXISTS secrets;
-- +goose StatementEnd
