-- +goose Up
CREATE TABLE registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    participant_id UUID NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    distance_id UUID NOT NULL REFERENCES distances(id) ON DELETE CASCADE,
    category VARCHAR(20) NOT NULL,
    bib_number VARCHAR(20) UNIQUE,
    qr_code TEXT,
    time VARCHAR(20),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'cancelled')),
    registration_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    checked_in_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(event_id, participant_id, distance_id)
);

CREATE INDEX idx_registrations_event_id ON registrations(event_id);
CREATE INDEX idx_registrations_participant_id ON registrations(participant_id);
CREATE INDEX idx_registrations_distance_id ON registrations(distance_id);
CREATE INDEX idx_registrations_status ON registrations(status);

-- +goose Down
DROP TABLE IF EXISTS registrations;
