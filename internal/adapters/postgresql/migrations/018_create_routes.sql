-- +goose Up
CREATE TABLE routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    distance_id UUID REFERENCES distances(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    geojson JSONB NOT NULL,
    elevation_profile JSONB,
    total_distance DECIMAL(10,2),
    total_elevation_gain DECIMAL(8,2),
    min_elevation DECIMAL(8,2),
    max_elevation DECIMAL(8,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_routes_event_id ON routes(event_id);
CREATE INDEX idx_routes_distance_id ON routes(distance_id);

-- +goose Down
DROP TABLE IF EXISTS routes;
