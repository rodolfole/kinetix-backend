-- +goose Up
CREATE TABLE age_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    min_age INTEGER,
    max_age INTEGER,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Seed data for Mexican running categories
INSERT INTO age_categories (name, min_age, max_age, display_order) VALUES
    ('Juvenil', 15, 17, 1),
    ('Libre', 18, 39, 2),
    ('Master 40-44', 40, 44, 3),
    ('Master 45-49', 45, 49, 4),
    ('Master 50-54', 50, 54, 5),
    ('Master 55-59', 55, 59, 6),
    ('Master 60-64', 60, 64, 7),
    ('Master 65-69', 65, 69, 8),
    ('Master 70-74', 70, 74, 9),
    ('Master 75-79', 75, 79, 10),
    ('Master 80+', 80, NULL, 11);

-- +goose Down
DROP TABLE IF EXISTS age_categories;
