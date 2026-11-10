-- +goose Up
-- +goose StatementBegin
-- Platform-level configuration. Singleton row (id=1) for the global commission %.
-- Updated_at + updated_by are auditable fields populated by the admin module.
CREATE TABLE platform_config (
    id              SERIAL PRIMARY KEY,
    commission_pct  NUMERIC(5,2) NOT NULL DEFAULT 10.00
        CHECK (commission_pct >= 0 AND commission_pct <= 100),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_by      TEXT
);
-- Seed the singleton row.
INSERT INTO platform_config DEFAULT VALUES;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS platform_config;
-- +goose StatementEnd
