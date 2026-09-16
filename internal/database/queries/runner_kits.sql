-- sqlc queries for runner_kit_items

-- name: CreateRunnerKitItem :one
INSERT INTO runner_kit_items (event_id, name, icon) VALUES ($1, $2, $3) RETURNING *;

-- name: GetRunnerKitItem :one
SELECT * FROM runner_kit_items WHERE id = $1 LIMIT 1;

-- name: ListRunnerKitItemsByEvent :many
SELECT * FROM runner_kit_items WHERE event_id = $1 ORDER BY name ASC;

-- name: UpdateRunnerKitItem :one
UPDATE runner_kit_items SET name = $2, icon = $3 WHERE id = $1 RETURNING *;

-- name: DeleteRunnerKitItem :exec
DELETE FROM runner_kit_items WHERE id = $1;
