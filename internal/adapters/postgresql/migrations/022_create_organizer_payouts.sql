-- +goose Up
-- +goose StatementBegin
-- Per-organizer pending/sent/failed payout log. One row is created on each
-- successful order webhook; admin can dispatch and mark sent.
-- amount_cents is the net after platform commission, in the same currency as the order.
CREATE TABLE organizer_payouts (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id     UUID         NOT NULL REFERENCES organizers(id) ON DELETE CASCADE,
    order_id         UUID         NOT NULL REFERENCES orders(id)    ON DELETE CASCADE,
    amount_cents     BIGINT       NOT NULL CHECK (amount_cents >= 0),
    currency         TEXT         NOT NULL DEFAULT 'MXN',
    commission_pct   NUMERIC(5,2) NOT NULL,
    status           TEXT         NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','sent','failed')),
    mp_payout_id     TEXT,
    failure_reason  TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    sent_at          TIMESTAMPTZ
);
CREATE INDEX idx_organizer_payouts_status      ON organizer_payouts(status);
CREATE INDEX idx_organizer_payouts_organizer  ON organizer_payouts(organizer_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_organizer_payouts_organizer;
DROP INDEX IF EXISTS idx_organizer_payouts_status;
DROP TABLE IF EXISTS organizer_payouts;
-- +goose StatementEnd
