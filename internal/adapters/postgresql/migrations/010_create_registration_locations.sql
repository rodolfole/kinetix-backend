-- +goose Up
CREATE TABLE registration_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    location_name VARCHAR(255) NOT NULL,
    address TEXT NOT NULL,
    schedule VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_registration_locations_event_id ON registration_locations(event_id);

-- +goose Down
DROP TABLE IF EXISTS registration_locations;
