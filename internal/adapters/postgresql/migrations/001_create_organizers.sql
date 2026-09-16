-- +goose Up
CREATE TABLE organizers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_name VARCHAR(255) NOT NULL,
    brand_name VARCHAR(255),
    rfc VARCHAR(13) UNIQUE NOT NULL,
    billing_zip_code VARCHAR(10) NOT NULL,
    billing_state VARCHAR(100) NOT NULL,
    billing_city VARCHAR(100) NOT NULL,
    contact_name VARCHAR(255) NOT NULL,
    contact_email VARCHAR(255) NOT NULL,
    contact_phone VARCHAR(20) NOT NULL,
    logo_url VARCHAR(500),
    is_active BOOLEAN DEFAULT TRUE,
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_organizers_rfc ON organizers(rfc);
CREATE INDEX idx_organizers_contact_email ON organizers(contact_email);

-- +goose Down
DROP TABLE IF EXISTS organizers;
