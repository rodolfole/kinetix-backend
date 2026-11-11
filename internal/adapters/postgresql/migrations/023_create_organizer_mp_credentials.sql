-- +goose Up
-- +goose StatementBegin
-- OAuth credentials for each organizer's MercadoPago Marketplace account.
-- Populated by /admin/organizers/:id/connect-mp OAuth callback.
-- Tokens are refreshed periodically by a cron job (out of scope of this slice).
CREATE TABLE organizer_mp_credentials (
    organizer_id    UUID         PRIMARY KEY REFERENCES organizers(id) ON DELETE CASCADE,
    mp_user_id      TEXT         NOT NULL,
    access_token    TEXT         NOT NULL,
    refresh_token   TEXT         NOT NULL,
    expires_at      TIMESTAMPTZ  NOT NULL,
    connected_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS organizer_mp_credentials;
-- +goose StatementEnd
