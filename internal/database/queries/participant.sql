-- sqlc queries for participants

-- name: CreateParticipant :one
INSERT INTO participants (
    first_name, last_name, second_last_name, gender, birth_date, email, 
    phone, country, municipality, state, zip_code, team, size, 
    emergency_contact_name, emergency_contact_phone, medical_info, waiver_signed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
) RETURNING *;

-- name: GetParticipant :one
SELECT * FROM participants
WHERE id = $1 LIMIT 1;

-- name: GetParticipantByEmail :one
SELECT * FROM participants
WHERE email = $1 LIMIT 1;

-- name: ListParticipants :many
SELECT * FROM participants
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateParticipant :one
UPDATE participants
SET 
    first_name = $2,
    last_name = $3,
    second_last_name = $4,
    gender = $5,
    birth_date = $6,
    email = $7,
    phone = $8,
    country = $9,
    municipality = $10,
    state = $11,
    zip_code = $12,
    team = $13,
    size = $14,
    emergency_contact_name = $15,
    emergency_contact_phone = $16,
    medical_info = $17,
    waiver_signed_at = $18,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteParticipant :exec
DELETE FROM participants
WHERE id = $1;
