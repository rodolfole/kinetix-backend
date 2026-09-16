-- +goose Up
CREATE TABLE distance_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL,
    km DECIMAL(5,2) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE distances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    distance_type_id UUID REFERENCES distance_types(id),
    km DECIMAL(5,2) NOT NULL,
    capacity INTEGER NOT NULL,
    surface VARCHAR(50) NOT NULL,
    elevation INTEGER DEFAULT NULL,
    time_limit VARCHAR(20) DEFAULT NULL,
    start_lat DECIMAL(10,8),
    start_lng DECIMAL(11,8),
    end_lat DECIMAL(10,8),
    end_lng DECIMAL(11,8),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_distances_event_id ON distances(event_id);
CREATE INDEX idx_distance_types_km ON distance_types(km);

-- Seed distance types
INSERT INTO distance_types (name, km) VALUES
    ('5K', 5),
    ('10K', 10),
    ('15K', 15),
    ('21K Half Marathon', 21.1),
    ('42K Marathon', 42.2);

-- +goose Down
DROP TABLE IF EXISTS distances;
DROP TABLE IF EXISTS distance_types;
