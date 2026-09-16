-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'organizer', 'participant')),
    organizer_id UUID REFERENCES organizers(id) ON DELETE SET NULL,
    participant_id UUID REFERENCES participants(id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT TRUE,
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_organizer_id ON users(organizer_id);
CREATE INDEX idx_users_participant_id ON users(participant_id);

-- +goose Down
DROP TABLE IF EXISTS users;