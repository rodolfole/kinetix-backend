-- sqlc queries for orders

-- name: CreateOrder :one
INSERT INTO orders (
    registration_id, participant_id, event_id, amount, currency, 
    status, payment_method, mercado_pago_payment_id, mercado_pago_preference_id, 
    transaction_date
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders
WHERE id = $1 LIMIT 1;

-- name: GetOrderByRegistrationId :one
SELECT * FROM orders
WHERE registration_id = $1 LIMIT 1;

-- name: GetOrderByMercadoPagoPaymentId :one
SELECT * FROM orders
WHERE mercado_pago_payment_id = $1 LIMIT 1;

-- name: ListOrdersByParticipant :many
SELECT * FROM orders
WHERE participant_id = $1
ORDER BY created_at DESC;

-- name: ListOrdersByEvent :many
SELECT * FROM orders
WHERE event_id = $1
ORDER BY created_at DESC;

-- name: ListOrdersByStatus :many
SELECT * FROM orders
WHERE status = $1
ORDER BY created_at DESC;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateOrderPaymentInfo :one
UPDATE orders
SET 
    payment_method = $2,
    mercado_pago_payment_id = $3,
    transaction_date = $4,
    status = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteOrder :exec
DELETE FROM orders
WHERE id = $1;
