-- +goose Up
-- Money is stored as integer cents (ADR-12, PLAN-16 T11).
-- Percentages (platform_config.commission_pct) keep NUMERIC(5,2).

-- orders: amount -> amount_cents
ALTER TABLE orders ADD COLUMN amount_cents BIGINT NOT NULL DEFAULT 0 CHECK (amount_cents >= 0);
UPDATE orders SET amount_cents = round(amount * 100);
ALTER TABLE orders DROP COLUMN amount;

-- pricing_stages: price -> price_cents
ALTER TABLE pricing_stages ADD COLUMN price_cents BIGINT NOT NULL DEFAULT 0 CHECK (price_cents >= 0);
UPDATE pricing_stages SET price_cents = round(price * 100);
ALTER TABLE pricing_stages DROP COLUMN price;

-- awards: prize_amount -> prize_cents
ALTER TABLE awards ADD COLUMN prize_cents BIGINT NOT NULL DEFAULT 0 CHECK (prize_cents >= 0);
UPDATE awards SET prize_cents = round(prize_amount * 100);
ALTER TABLE awards DROP COLUMN prize_amount;

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders ADD COLUMN amount DECIMAL(10,2) NOT NULL DEFAULT 0;
UPDATE orders SET amount = amount_cents / 100.0;
ALTER TABLE orders DROP COLUMN amount_cents;

ALTER TABLE pricing_stages ADD COLUMN price DECIMAL(10,2) NOT NULL DEFAULT 0;
UPDATE pricing_stages SET price = price_cents / 100.0;
ALTER TABLE pricing_stages DROP COLUMN price_cents;

ALTER TABLE awards ADD COLUMN prize_amount DECIMAL(10,2) NOT NULL DEFAULT 0;
UPDATE awards SET prize_amount = prize_cents / 100.0;
ALTER TABLE awards DROP COLUMN prize_cents;
-- +goose StatementEnd
