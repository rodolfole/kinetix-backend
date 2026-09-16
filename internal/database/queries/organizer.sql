-- sqlc queries for organizers

-- name: CreateOrganizer :one
INSERT INTO organizers (
    business_name, brand_name, rfc, billing_zip_code, billing_state, billing_city, 
    contact_email, contact_phone
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetOrganizer :one
SELECT * FROM organizers
WHERE id = $1 LIMIT 1;

-- name: GetOrganizerByRFC :one
SELECT * FROM organizers
WHERE rfc = $1 LIMIT 1;

-- name: ListOrganizers :many
SELECT * FROM organizers
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateOrganizer :one
UPDATE organizers
SET 
    business_name = $2,
    brand_name = $3,
    billing_zip_code = $4,
    billing_state = $5,
    billing_city = $6,
    contact_email = $7,
    contact_phone = $8,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteOrganizer :exec
DELETE FROM organizers
WHERE id = $1;
