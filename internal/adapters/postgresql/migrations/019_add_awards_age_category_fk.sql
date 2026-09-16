-- +goose Up
ALTER TABLE awards ADD CONSTRAINT fk_awards_age_category
    FOREIGN KEY (age_category_id) REFERENCES age_categories(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE awards DROP CONSTRAINT IF EXISTS fk_awards_age_category;
