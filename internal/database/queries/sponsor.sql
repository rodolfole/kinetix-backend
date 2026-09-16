-- sqlc queries for sponsors

-- name: CreateSponsor :one
INSERT INTO sponsors (
    event_id, name, tier, logo_url, website_url
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetSponsor :one
SELECT * FROM sponsors
WHERE id = $1 LIMIT 1;

-- name: ListSponsorsByEvent :many
SELECT * FROM sponsors
WHERE event_id = $1
ORDER BY tier;

-- name: UpdateSponsor :one
UPDATE sponsors
SET name = $2, tier = $3, logo_url = $4, website_url = $5
WHERE id = $1
RETURNING *;

-- name: DeleteSponsor :exec
DELETE FROM sponsors
WHERE id = $1;
