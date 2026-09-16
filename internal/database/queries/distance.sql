-- sqlc queries for distances

-- name: CreateDistance :one
INSERT INTO distances (
    event_id, km, capacity, surface, start_lat, start_lng, end_lat, end_lng
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetDistance :one
SELECT * FROM distances
WHERE id = $1 LIMIT 1;

-- name: ListDistancesByEvent :many
SELECT * FROM distances
WHERE event_id = $1
ORDER BY km ASC;

-- name: UpdateDistance :one
UPDATE distances
SET km = $2, capacity = $3, surface = $4, start_lat = $5, start_lng = $6, end_lat = $7, end_lng = $8
WHERE id = $1
RETURNING *;

-- name: DeleteDistance :exec
DELETE FROM distances
WHERE id = $1;
