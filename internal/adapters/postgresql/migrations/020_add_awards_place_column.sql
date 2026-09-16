-- +goose Up
-- Migration 004 declares `place` but on some existing environments the
-- column was never created (the live DB only has id, distance_id,
-- age_category_id, gender, prize_amount, created_at). The Go code
-- (CreateAwardParams.Place) and the unique index uq_awards_distance_age_gender_place
-- both reference `place`, so INSERTs were failing with
-- `column "place" of relation "awards" does not exist`.
--
-- This migration is idempotent (ADD COLUMN IF NOT EXISTS) so it's safe
-- to re-run on a DB that already has the column. We also re-create the
-- unique index in case it was dropped along with the column.

ALTER TABLE awards
    ADD COLUMN IF NOT EXISTS place INT NOT NULL DEFAULT 1 CHECK (place >= 1);

CREATE UNIQUE INDEX IF NOT EXISTS uq_awards_distance_age_gender_place
    ON awards(distance_id, age_category_id, gender, place);

-- +goose Down
DROP INDEX IF EXISTS uq_awards_distance_age_gender_place;
ALTER TABLE awards DROP COLUMN IF EXISTS place;
