-- sqlc queries for registration locations

-- name: CreateRegistrationLocation :one
INSERT INTO registration_locations (
    event_id, location_name, address, schedule
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetRegistrationLocation :one
SELECT * FROM registration_locations
WHERE id = $1 LIMIT 1;

-- name: ListRegistrationLocationsByEvent :many
SELECT * FROM registration_locations
WHERE event_id = $1
ORDER BY location_name;

-- name: UpdateRegistrationLocation :one
UPDATE registration_locations
SET location_name = $2, address = $3, schedule = $4
WHERE id = $1
RETURNING *;

-- name: DeleteRegistrationLocation :exec
DELETE FROM registration_locations
WHERE id = $1;
