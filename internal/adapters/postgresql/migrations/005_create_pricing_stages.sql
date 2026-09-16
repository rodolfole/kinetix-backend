-- +goose Up
CREATE TABLE pricing_stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    distance_id UUID NOT NULL REFERENCES distances(id) ON DELETE CASCADE,
    name VARCHAR(100),
    price DECIMAL(10,2) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_early_bird BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT valid_date_range CHECK (end_date >= start_date)
);

CREATE INDEX idx_pricing_distance_id ON pricing_stages(distance_id);
CREATE INDEX idx_pricing_dates ON pricing_stages(start_date, end_date);

-- +goose Down
DROP TABLE IF EXISTS pricing_stages;
