-- sqlc queries for organizers

-- name: CreateOrganizer :one
INSERT INTO organizers (
    business_name, brand_name, rfc, billing_zip_code, billing_state, billing_city,
    contact_name, contact_email, contact_phone, logo_url
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
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
    contact_name = $7,
    contact_email = $8,
    contact_phone = $9,
    logo_url = $10,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteOrganizer :exec
DELETE FROM organizers
WHERE id = $1;


-- sqlc queries for events

-- name: CreateEvent :one
INSERT INTO events (
    organizer_id, name, slug, description, event_date, deadline,
    state, municipality, address, logo_url, judges, rules, risks, transit, itinerary, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
) RETURNING *;

-- name: GetEvent :one
SELECT * FROM events
WHERE id = $1 LIMIT 1;

-- name: GetEventBySlug :one
SELECT * FROM events
WHERE slug = $1 LIMIT 1;

-- name: ListEventsByOrganizer :many
SELECT * FROM events
WHERE organizer_id = $1
ORDER BY event_date DESC;

-- name: ListEvents :many
SELECT * FROM events
WHERE status = COALESCE($1, status)
ORDER BY event_date DESC
LIMIT $2 OFFSET $3;

-- name: UpdateEvent :one
UPDATE events
SET
    name = $2,
    slug = $3,
    description = $4,
    event_date = $5,
    deadline = $6,
    state = $7,
    municipality = $8,
    address = $9,
    logo_url = $10,
    judges = $11,
    rules = $12,
    risks = $13,
    transit = $14,
    itinerary = $15,
    status = $16,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateEventStatus :one
UPDATE events
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteEvent :exec
DELETE FROM events
WHERE id = $1;


-- sqlc queries for distance_types

-- name: CreateDistanceType :one
INSERT INTO distance_types (name, km) VALUES ($1, $2) RETURNING *;

-- name: GetDistanceType :one
SELECT * FROM distance_types WHERE id = $1 LIMIT 1;

-- name: GetDistanceTypeByKM :one
SELECT * FROM distance_types WHERE km = $1 LIMIT 1;

-- name: ListDistanceTypes :many
SELECT * FROM distance_types ORDER BY km ASC;

-- name: DeleteDistanceType :exec
DELETE FROM distance_types WHERE id = $1;


-- sqlc queries for distances

-- name: CreateDistance :one
INSERT INTO distances (
    event_id, distance_type_id, km, capacity, surface, elevation, time_limit, start_lat, start_lng, end_lat, end_lng
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetDistance :one
SELECT * FROM distances
WHERE id = $1 LIMIT 1;

-- name: GetDistanceByEventAndType :one
SELECT * FROM distances
WHERE event_id = $1 AND distance_type_id = $2
LIMIT 1;

-- name: ListDistancesByEvent :many
SELECT * FROM distances
WHERE event_id = $1
ORDER BY km ASC;

-- name: UpdateDistance :one
UPDATE distances
SET distance_type_id = $2, km = $3, capacity = $4, surface = $5, elevation = $6, time_limit = $7, start_lat = $8, start_lng = $9, end_lat = $10, end_lng = $11
WHERE id = $1
RETURNING *;

-- name: DeleteDistance :exec
DELETE FROM distances
WHERE id = $1;


-- sqlc queries for awards

-- name: CreateAward :one
INSERT INTO awards (
    distance_id, age_category_id, gender, prize_amount
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetAward :one
SELECT * FROM awards
WHERE id = $1 LIMIT 1;

-- name: ListAwardsByDistance :many
SELECT * FROM awards
WHERE distance_id = $1
ORDER BY age_category_id;

-- name: UpdateAward :one
UPDATE awards
SET age_category_id = $2, gender = $3, prize_amount = $4
WHERE id = $1
RETURNING *;

-- name: DeleteAward :exec
DELETE FROM awards
WHERE id = $1;


-- sqlc queries for pricing stages

-- name: CreatePricingStage :one
INSERT INTO pricing_stages (
    distance_id, name, price, start_date, end_date, is_early_bird
) VALUES (
    $1, $2, $3, $4, $5, $6
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
SET name = $2, price = $3, start_date = $4, end_date = $5, is_early_bird = $6
WHERE id = $1
RETURNING *;

-- name: DeletePricingStage :exec
DELETE FROM pricing_stages
WHERE id = $1;


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


-- sqlc queries for registrations

-- name: GetMaxBibNumberByEvent :one
SELECT COALESCE(MAX(SUBSTRING(bib_number FROM '^[0-9]+$')::INTEGER), 0) as max_num
FROM registrations
WHERE event_id = $1 AND bib_number IS NOT NULL;

-- name: CreateRegistration :one
INSERT INTO registrations (
    event_id, participant_id, distance_id, category, bib_number, qr_code, time, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
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

-- name: UpdateRegistrationQRCode :one
UPDATE registrations
SET qr_code = $2
WHERE id = $1
RETURNING *;

-- name: UpdateRegistrationCheckIn :one
UPDATE registrations
SET checked_in_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetRegistrationByBibNumber :one
SELECT * FROM registrations
WHERE event_id = $1 AND bib_number = $2 LIMIT 1;

-- name: DeleteRegistration :exec
DELETE FROM registrations
WHERE id = $1;


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


-- sqlc queries for users

-- name: CreateUser :one
INSERT INTO users (
    email, password_hash, role, organizer_id, participant_id
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByOrganizerID :one
SELECT * FROM users
WHERE organizer_id = $1 LIMIT 1;

-- name: GetUserByParticipantID :one
SELECT * FROM users
WHERE participant_id = $1 LIMIT 1;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserLastLogin :one
UPDATE users
SET last_login = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserActive :one
UPDATE users
SET is_active = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;


-- sqlc queries for event_participant_fields

-- name: CreateEventParticipantField :one
INSERT INTO event_participant_fields (
    event_id, field_name, is_required, is_enabled, display_order
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetEventParticipantField :one
SELECT * FROM event_participant_fields
WHERE id = $1 LIMIT 1;

-- name: ListEventParticipantFieldsByEvent :many
SELECT * FROM event_participant_fields
WHERE event_id = $1
ORDER BY display_order ASC;

-- name: UpsertEventParticipantField :one
INSERT INTO event_participant_fields (
    event_id, field_name, is_required, is_enabled, display_order
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (event_id, field_name)
DO UPDATE SET
    is_required = $3,
    is_enabled = $4,
    display_order = $5
RETURNING *;

-- name: DeleteEventParticipantFieldsByEvent :exec
DELETE FROM event_participant_fields
WHERE event_id = $1;


-- sqlc queries for event_field_config

-- name: ListEventFieldConfigByEvent :many
SELECT * FROM event_field_config
WHERE event_id = $1
ORDER BY section_name ASC;

-- name: UpsertEventFieldConfig :one
INSERT INTO event_field_config (
    event_id, section_name, is_enabled, is_required
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (event_id, section_name)
DO UPDATE SET
    is_enabled = $3,
    is_required = $4,
    updated_at = NOW()
RETURNING *;

-- name: DeleteEventFieldConfigByEvent :exec
DELETE FROM event_field_config
WHERE event_id = $1;


-- sqlc queries for age_categories

-- name: ListAgeCategories :many
SELECT * FROM age_categories ORDER BY display_order ASC;


-- sqlc queries for event_tags

-- name: CreateEventTag :one
INSERT INTO event_tags (event_id, name) VALUES ($1, $2) RETURNING *;

-- name: GetEventTag :one
SELECT * FROM event_tags WHERE id = $1 LIMIT 1;

-- name: ListEventTagsByEvent :many
SELECT * FROM event_tags WHERE event_id = $1 ORDER BY name ASC;

-- name: ListEventTagsByNames :many
SELECT * FROM event_tags WHERE event_id = $1 AND name = ANY($2::varchar(50)[]) ORDER BY name ASC;

-- name: DeleteEventTag :exec
DELETE FROM event_tags WHERE id = $1;

-- name: DeleteEventTagsByEvent :exec
DELETE FROM event_tags WHERE event_id = $1;

-- name: DeleteEventTagsByNames :exec
DELETE FROM event_tags WHERE event_id = sqlc.arg(event_id) AND name = ANY(sqlc.arg(names)::varchar(50)[]);


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

-- sqlc queries for routes

-- name: CreateRoute :one
INSERT INTO routes (
    event_id, distance_id, name, geojson, elevation_profile,
    total_distance, total_elevation_gain, min_elevation, max_elevation
) VALUES (
    sqlc.arg(event_id), sqlc.arg(distance_id), sqlc.arg(name), sqlc.arg(geojson), sqlc.arg(elevation_profile),
    sqlc.arg(total_distance), sqlc.arg(total_elevation_gain), sqlc.arg(min_elevation), sqlc.arg(max_elevation)
) RETURNING *;

-- name: GetRoute :one
SELECT * FROM routes WHERE id = sqlc.arg(id) LIMIT 1;

-- name: ListRoutesByEvent :many
SELECT * FROM routes WHERE event_id = sqlc.arg(event_id) ORDER BY name ASC;

-- name: ListRoutesByDistance :many
SELECT * FROM routes WHERE distance_id = sqlc.arg(distance_id) ORDER BY name ASC;

-- name: DeleteRoute :exec
DELETE FROM routes WHERE id = sqlc.arg(id);