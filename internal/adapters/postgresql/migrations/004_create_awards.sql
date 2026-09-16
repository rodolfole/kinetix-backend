-- +goose Up
CREATE TABLE awards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    distance_id UUID NOT NULL REFERENCES distances(id) ON DELETE CASCADE,
    age_category_id UUID NOT NULL,
    gender VARCHAR(20) NOT NULL CHECK (gender IN ('male', 'female', 'general')),
    -- Posición (1 = primer lugar, 2 = segundo, etc). Default 1 mantiene
    -- retro-compat si el caller omite el campo.
    place INT NOT NULL DEFAULT 1 CHECK (place >= 1),
    prize_amount DECIMAL(10,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_awards_distance_id ON awards(distance_id);
-- Unicidad lógica: no puede haber dos premiaciones para el mismo
-- (distancia, categoría de edad, género, lugar).
CREATE UNIQUE INDEX uq_awards_distance_age_gender_place
    ON awards(distance_id, age_category_id, gender, place);

-- +goose Down
DROP TABLE IF EXISTS awards;
