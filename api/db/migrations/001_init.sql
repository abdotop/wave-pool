-- +goose Up
-- +goose StatementBegin
CREATE TABLE "users" (
    "id" char(27) PRIMARY KEY,
    "phone" varchar(20) UNIQUE NOT NULL,
    "pin_hash" varchar(128) NOT NULL,
    "status" varchar(16) NOT NULL DEFAULT 'active',
    "created_at" timestamptz DEFAULT now()
);

CREATE TABLE "business" (
    "id" char(27) PRIMARY KEY,
    "name" text NOT NULL,
    "country" char(2) NOT NULL,
    "currency" char(3) NOT NULL,
    "created_at" timestamptz DEFAULT now()
);

CREATE TABLE "api_keys" (
    "id" char(27) PRIMARY KEY,
    "business_id" char(27) NOT NULL REFERENCES business(id),
    "prefix" varchar(32) NOT NULL,
    "key_hash" varchar(128) NOT NULL,
    "scopes" text[] NOT NULL,
    "env" varchar(16) NOT NULL,
    "status" varchar(16) DEFAULT 'active',
    "created_at" timestamptz DEFAULT now()
);

CREATE TABLE "webhooks" (
    "id" char(27) PRIMARY KEY,
    "business_id" char(27) NOT NULL REFERENCES business(id),
    "url" text NOT NULL,
    "strategy" varchar(32) NOT NULL,
    "secret" varchar(128) NOT NULL,
    "events" text[] NOT NULL,
    "status" varchar(16) DEFAULT 'active',
    "created_at" timestamptz DEFAULT now()
);

CREATE TABLE "checkout_sessions" (
    "id" char(27) PRIMARY KEY,
    "business_id" char(27) NOT NULL REFERENCES business(id),
    "amount" varchar(32) NOT NULL,
    "currency" char(3) NOT NULL,
    "client_reference" varchar(255),
    "status" varchar(16) NOT NULL,
    "error_url" text NOT NULL,
    "success_url" text NOT NULL,
    "restrict_payer_mobile" varchar(20),
    "wave_launch_url" text,
    "transaction_id" varchar(32),
    "payment_status" varchar(16),
    "last_payment_error" jsonb,
    "expires_at" timestamptz NOT NULL,
    "when_completed" timestamptz,
    "when_created" timestamptz DEFAULT now()
);

CREATE TABLE "payments" (
    "id" char(27) PRIMARY KEY,
    "session_id" char(27) NOT NULL REFERENCES checkout_sessions(id),
    "amount" varchar(32) NOT NULL,
    "currency" char(3) NOT NULL,
    "status" varchar(16) NOT NULL,
    "failure_reason" text,
    "completed_at" timestamptz,
    "created_at" timestamptz DEFAULT now()
);

CREATE TABLE "webhook_deliveries" (
    "id" char(27) PRIMARY KEY,
    "webhook_id" char(27) NOT NULL REFERENCES webhooks(id),
    "event_type" text NOT NULL,
    "payload" jsonb NOT NULL,
    "status" varchar(16) NOT NULL,
    "http_code" integer,
    "attempts" integer DEFAULT 0,
    "last_error" text,
    "sent_at" timestamptz,
    "response_ms" integer
);

CREATE TABLE "b2b_transfers" (
    "id" char(27) PRIMARY KEY,
    "business_id" char(27) NOT NULL REFERENCES business(id),
    "counterparty" char(27) NOT NULL,
    "amount" varchar(32) NOT NULL,
    "currency" char(3) NOT NULL,
    "status" varchar(16) NOT NULL,
    "reference" varchar(255),
    "created_at" timestamptz DEFAULT now()
);

CREATE TABLE "balances" (
    "business_id" char(27) PRIMARY KEY REFERENCES business(id),
    "available" varchar(32) NOT NULL,
    "pending" varchar(32) NOT NULL,
    "currency" char(3) NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "balances";
DROP TABLE IF EXISTS "b2b_transfers";
DROP TABLE IF EXISTS "webhook_deliveries";
DROP TABLE IF EXISTS "payments";
DROP TABLE IF EXISTS "checkout_sessions";
DROP TABLE IF EXISTS "webhooks";
DROP TABLE IF EXISTS "api_keys";
DROP TABLE IF EXISTS "business";
DROP TABLE IF EXISTS "users";
-- +goose StatementEnd
