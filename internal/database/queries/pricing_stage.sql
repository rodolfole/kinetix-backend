-- sqlc queries for pricing stages

-- name: CreatePricingStage :one
INSERT INTO pricing_stages (
    distance_id, name, price, start_date, end_date
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetPricingStage :one
SELECT * FROM pricing_stages
WHERE id = $1 LIMIT 1;

-- name: ListPricingStagesByDistance :many
SELECT * FROM pricing_stages
WHERE distance_id = $1
ORDER BY start_date ASC;

-- name: GetCurrentPricing :many
SELECT * FROM pricing_stages
WHERE distance_id = $1 
  AND start_date <= CURRENT_DATE 
  AND end_date >= CURRENT_DATE
ORDER BY price ASC;

-- name: UpdatePricingStage :one
UPDATE pricing_stages
SET name = $2, price = $3, start_date = $4, end_date = $5
WHERE id = $1
RETURNING *;

-- name: DeletePricingStage :exec
DELETE FROM pricing_stages
WHERE id = $1;
