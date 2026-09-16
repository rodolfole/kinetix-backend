-- +goose Up
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id UUID NOT NULL REFERENCES organizers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    event_date TIMESTAMP WITH TIME ZONE NOT NULL,
    deadline DATE NOT NULL,
    state VARCHAR(100) NOT NULL,
    municipality VARCHAR(100) NOT NULL,
    address TEXT NOT NULL,
    logo_url TEXT NOT NULL DEFAULT '',
    judges TEXT,
    rules TEXT,
    risks TEXT,
    transit TEXT,
    itinerary JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'cancelled', 'completed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_events_organizer_id ON events(organizer_id);
CREATE INDEX idx_events_slug ON events(slug);
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_events_event_date ON events(event_date);

-- +goose Down
DROP TABLE IF EXISTS events;
