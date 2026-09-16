-- +goose Up
CREATE TABLE sponsors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    tier VARCHAR(50) NOT NULL,
    logo_url TEXT,
    website_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_sponsors_event_id ON sponsors(event_id);

-- +goose Down
DROP TABLE IF EXISTS sponsors;
