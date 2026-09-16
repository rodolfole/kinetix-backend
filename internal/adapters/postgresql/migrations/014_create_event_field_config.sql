-- +goose Up
-- Configurable fields for event organizer form (toggleable sections)
CREATE TABLE event_field_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    section_name VARCHAR(50) NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT false,
    is_required BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(event_id, section_name)
);

CREATE INDEX idx_event_field_config_event_id ON event_field_config(event_id);

-- +goose Down
DROP TABLE IF EXISTS event_field_config;
