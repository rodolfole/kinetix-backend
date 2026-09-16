-- sqlc queries for awards

-- name: CreateAward :one
INSERT INTO awards (
    distance_id, age_category_id, gender, place, prize_amount
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetAward :one
SELECT * FROM awards
WHERE id = $1 LIMIT 1;

-- name: ListAwardsByDistance :many
SELECT * FROM awards
WHERE distance_id = $1
ORDER BY age_category_id, place;

-- name: UpdateAward :one
UPDATE awards
SET age_category_id = $2, gender = $3, place = $4, prize_amount = $5
WHERE id = $1
RETURNING *;

-- name: DeleteAward :exec
DELETE FROM awards
WHERE id = $1;
