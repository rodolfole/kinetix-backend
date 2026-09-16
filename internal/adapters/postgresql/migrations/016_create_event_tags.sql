-- +goose Up
CREATE TABLE event_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_event_tags_event_id ON event_tags(event_id);

-- +goose Down
DROP TABLE IF EXISTS event_tags;
