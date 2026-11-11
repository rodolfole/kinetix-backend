//go:build integration

// TestWebhookIdempotencyStorage pins migration 024's unique indexes:
// one MercadoPago payment id per order and one payout per order.
package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getenvDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// miniFixtures creates the minimum rows the webhook storage needs: one
// organizer, one event, one participant, one registration, and cleans after.
func miniFixtures(ctx context.Context, t *testing.T, pool *pgxpool.Pool) (organizerID, eventID, participantID, registrationID uuid.UUID) {
	t.Helper()

	if err := pool.QueryRow(ctx, `
insert into organizers (business_name, brand_name, rfc, billing_zip_code, billing_state, billing_city, contact_name, contact_email, contact_phone)
values ('TEST Organizer i', 'Kineticos', 'ABC123456XY1', '44500', 'CDMX', 'CDMX', 'Ana', 'idem-test@kinetix.test', '5550001111')
returning id`).Scan(&organizerID); err != nil {
		t.Fatalf("organizer fixture: %v", err)
	}

	if err := pool.QueryRow(ctx, `
insert into events (organizer_id, name, slug, description, status, event_date, state)
values ($1, 'Idea Test', 'idem-test-' || replace(cast(gen_random_uuid() as text),'-',''), 'n', 'draft', now(), 'CDMX')
returning id`, organizerID).Scan(&eventID); err != nil {
		t.Fatalf("event fixture: %v", err)
	}

	if err := pool.QueryRow(ctx, `
insert into participants (first_name, last_name, email, birth_date, gender, phone, state)
values ('Pedro', 'Test', 'idem-test@kinetix.test', now()- interval '26 years', 'male', '5551234567', 'CDMX')
returning id`).Scan(&participantID); err != nil {
		t.Fatalf("participant fixture: %v", err)
	}

	var distanceID uuid.UUID
	if err := pool.QueryRow(ctx, `
insert into distances (event_id, km, capacity, surface)
values ($1, 5, 100, 'road')
returning id`, eventID).Scan(&distanceID); err != nil {
		t.Fatalf("distance fixture: %v", err)
	}
	_ = distanceID

	if err := pool.QueryRow(ctx, `
insert into registrations (event_id, participant_id, distance_id, status, registration_date)
values ($1, $2, $3, 'pending', now())
returning id`, eventID, participantID, distanceID).Scan(&registrationID); err != nil {
		t.Fatalf("registration fixture: %v", err)
	}

	t.Cleanup(func() {
		ctx2, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _ = pool.Exec(ctx2, `delete from orders where registration_id = $1`, registrationID)
		var unused uuid.UUID
		_ = unused
		_, _ = pool.Exec(ctx2, `delete from registrations where id = $1`, registrationID)
		_, _ = pool.Exec(ctx2, `delete from participants where id = $1`, participantID)
		_, _ = pool.Exec(ctx2, `delete from events where id = $1`, eventID)
		_, _ = pool.Exec(ctx2, `delete from organizers where id = $1`, organizerID)
	})
	return organizerID, eventID, participantID, registrationID
}

func TestWebhookIdempotencyStorage(t *testing.T) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		getenvDefault("DB_USER", "postgres"), getenvDefault("DB_PASSWORD", "postgres"),
		getenvDefault("DB_HOST", "localhost"), getenvDefault("DB_PORT", "5432"),
		getenvDefault("DB_NAME", "kinetix-db"), getenvDefault("DB_SSLMODE", "disable"))

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	organizerID, eventID, participantID, registrationID := miniFixtures(ctx, t, pool)

	payID := "testpay-" + uuid.New().String()
	orderID := uuid.New()

	if _, err := pool.Exec(ctx, `
insert into orders (id, registration_id, participant_id, event_id, amount, currency, status, mercado_pago_payment_id, transaction_date)
values ($1,$2,$3,$4, 100, 'MXN', 'pending', $5, now())`, orderID, registrationID, participantID, eventID, payID); err != nil {
		t.Fatalf("first order insert: %v", err)
	}
	if _, err := pool.Exec(ctx, `
insert into orders (id, registration_id, participant_id, event_id, amount, currency, status, mercado_pago_payment_id, transaction_date)
values (gen_random_uuid(), $1,$2,$3, 100, 'MXN', 'pending', $4, now())`, registrationID, participantID, eventID, payID); err == nil {
		t.Fatal("a duplicate mercado_pago_payment_id must violate the unique index")
	}
	if _, err := pool.Exec(ctx, `
insert into orders (id, registration_id, participant_id, event_id, amount, currency, status, mercado_pago_payment_id, transaction_date)
values (gen_random_uuid(), $1,$2,$3, 100, 'MXN', 'pending', NULL, now())`, registrationID, participantID, eventID); err != nil {
		t.Fatalf("orders without a payment id stay allowed: %v", err)
	}

	// Exactly one payout per order.
	if _, err := pool.Exec(ctx, `
insert into organizer_payouts (id, organizer_id, order_id, amount_cents, currency, commission_pct, status)
values (gen_random_uuid(), $1,$2, 9000, 'MXN', 10, 'pending')`, organizerID, orderID); err != nil {
		t.Fatalf("first payout insert: %v", err)
	}
	if _, err := pool.Exec(ctx, `
insert into organizer_payouts (id, organizer_id, order_id, amount_cents, currency, commission_pct, status)
values (gen_random_uuid(), $1,$2, 9000, 'MXN', 10, 'pending')`, organizerID, orderID); err == nil {
		t.Fatal("a duplicate payout per order must violate the unique index")
	}
}
