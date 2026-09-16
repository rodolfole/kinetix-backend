-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';
-- +goose StatementEnd

-- Apply trigger to organizers
CREATE TRIGGER update_organizers_updated_at
    BEFORE UPDATE ON organizers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to events
CREATE TRIGGER update_events_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to participants
CREATE TRIGGER update_participants_updated_at
    BEFORE UPDATE ON participants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to orders
CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS update_organizers_updated_at ON organizers;
DROP TRIGGER IF EXISTS update_events_updated_at ON events;
DROP TRIGGER IF EXISTS update_participants_updated_at ON participants;
DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;
DROP FUNCTION IF EXISTS update_updated_at_column();
