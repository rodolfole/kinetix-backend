-- +goose Up
CREATE TABLE event_participant_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    field_name VARCHAR(50) NOT NULL,
    is_required BOOLEAN NOT NULL DEFAULT false,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    display_order INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(event_id, field_name)
);

CREATE INDEX idx_event_participant_fields_event_id ON event_participant_fields(event_id);

-- +goose Down
DROP TABLE IF EXISTS event_participant_fields;
