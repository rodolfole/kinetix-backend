-- +goose Up
CREATE TABLE runner_kit_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    icon VARCHAR(50) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_runner_kit_items_event_id ON runner_kit_items(event_id);

-- +goose Down
DROP TABLE IF EXISTS runner_kit_items;
