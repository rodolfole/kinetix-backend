-- sqlc queries for events

-- name: CreateEvent :one
INSERT INTO events (
    organizer_id, name, slug, description, event_date, deadline, 
    state, municipality, address, logo_url, judges, rules, risks, transit, itinerary, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
) RETURNING *;

-- name: GetEvent :one
SELECT * FROM events
WHERE id = $1 LIMIT 1;

-- name: GetEventBySlug :one
SELECT * FROM events
WHERE slug = $1 LIMIT 1;

-- name: ListEventsByOrganizer :many
SELECT * FROM events
WHERE organizer_id = $1
ORDER BY event_date DESC;

-- name: ListEvents :many
SELECT * FROM events
WHERE status = COALESCE($1, status)
ORDER BY event_date DESC
LIMIT $2 OFFSET $3;

-- name: UpdateEvent :one
UPDATE events
SET 
    name = $2,
    slug = $3,
    description = $4,
    event_date = $5,
    deadline = $6,
    state = $7,
    municipality = $8,
    address = $9,
    logo_url = $10,
    judges = $11,
    rules = $12,
    risks = $13,
    transit = $14,
    itinerary = $15,
    status = $16,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateEventStatus :one
UPDATE events
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteEvent :exec
DELETE FROM events
WHERE id = $1;


-- sqlc queries for event_field_config

-- name: ListEventFieldConfigByEvent :many
SELECT * FROM event_field_config
WHERE event_id = $1
ORDER BY section_name ASC;

-- name: UpsertEventFieldConfig :one
INSERT INTO event_field_config (
    event_id, section_name, is_enabled, is_required
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (event_id, section_name)
DO UPDATE SET
    is_enabled = $3,
    is_required = $4,
    updated_at = NOW()
RETURNING *;

-- name: DeleteEventFieldConfigByEvent :exec
DELETE FROM event_field_config
WHERE event_id = $1;
