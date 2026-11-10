// payouts-cron is a one-shot job that finds all pending organizer_payouts,
// calls MP to disburse each, and updates the rows to sent (or failed).
//
// Run via: go run ./cmd/payouts-cron
// Or schedule with cron / Task Scheduler / k8s CronJob.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"kinetix-api/internal/admin"
	"kinetix-api/internal/config"
	"kinetix-api/internal/database"
	"kinetix-api/internal/mp"
	"kinetix-api/internal/store"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	cfg := config.Load()

	pool, err := database.NewConnectionPool(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		log.Fatalf("db pool: %v", err)
	}
	defer database.ClosePool(pool)
	appStore := store.NewStore(pool)

	mpClient := mp.NewClient(cfg.MercadoPago)
	adminRepo := admin.NewRepository(appStore)
	adminSvc := admin.NewService(adminRepo)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	rows, err := adminSvc.ListPayouts(ctx, "pending", 100)
	if err != nil {
		log.Fatalf("list pending payouts: %v", err)
	}

	log.Printf("found %d pending payouts", len(rows))
	successes, failures := 0, 0
	for _, p := range rows {
		if err := adminSvc.SendPayout(ctx, mpClient, p.ID); err != nil {
			log.Printf("FAIL payout %s: %v", p.ID, err)
			failures++
			continue
		}
		log.Printf("OK payout %s → amount_cents=%d", p.ID, p.AmountCents)
		successes++
	}

	log.Printf("done. successes=%d failures=%d", successes, failures)
	if failures > 0 {
		os.Exit(1)
	}
}
