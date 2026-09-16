-- +goose Up
CREATE TABLE participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    second_last_name VARCHAR(100),
    gender VARCHAR(20) NOT NULL,
    birth_date DATE NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    country VARCHAR(2) NOT NULL DEFAULT 'MX',
    municipality VARCHAR(100),
    state VARCHAR(100),
    zip_code VARCHAR(10),
    team VARCHAR(255),
    size VARCHAR(10),
    emergency_contact_name VARCHAR(255),
    emergency_contact_phone VARCHAR(20),
    medical_info TEXT,
    waiver_signed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_participants_email ON participants(email);
CREATE INDEX idx_participants_phone ON participants(phone);

-- +goose Down
DROP TABLE IF EXISTS participants;
