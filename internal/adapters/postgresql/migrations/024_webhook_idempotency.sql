-- +goose Up
-- +goose StatementBegin
-- Webhook idempotency at the storage level (PLAN-16 T10):
-- a MercadoPago payment id maps to exactly one order and one payout.
CREATE UNIQUE INDEX uq_orders_mp_payment_id
    ON orders (mercado_pago_payment_id)
    WHERE mercado_pago_payment_id <> '';

CREATE UNIQUE INDEX uq_organizer_payouts_order
    ON organizer_payouts (order_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uq_organizer_payouts_order;
DROP INDEX IF EXISTS uq_orders_mp_payment_id;
-- +goose StatementEnd
