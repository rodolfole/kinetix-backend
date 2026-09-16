-- sqlc queries for registrations

-- name: CreateRegistration :one
INSERT INTO registrations (
    event_id, participant_id, distance_id, category, time, status
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetRegistration :one
SELECT * FROM registrations
WHERE id = $1 LIMIT 1;

-- name: GetRegistrationByEventParticipant :one
SELECT * FROM registrations
WHERE event_id = $1 AND participant_id = $2 LIMIT 1;

-- name: ListRegistrationsByEvent :many
SELECT * FROM registrations
WHERE event_id = $1
ORDER BY registration_date DESC;

-- name: ListRegistrationsByParticipant :many
SELECT * FROM registrations
WHERE participant_id = $1
ORDER BY registration_date DESC;

-- name: UpdateRegistrationStatus :one
UPDATE registrations
SET status = $2
WHERE id = $1
RETURNING *;

-- name: DeleteRegistration :exec
DELETE FROM registrations
WHERE id = $1;
