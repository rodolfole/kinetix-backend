package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	host := env("DB_HOST", "localhost")
	port := env("DB_PORT", "5432")
	user := env("DB_USER", "postgres")
	password := env("DB_PASSWORD", "")
	dbName := env("DB_NAME", "kinetix-db")

	// Connect to default postgres database using pgx
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable", user, password, host, port)
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Error parsing config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer pool.Close()

	// Check if database exists
	var exists bool
	err = pool.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		log.Fatalf("Error checking database: %v", err)
	}

	if exists {
		fmt.Printf("Database '%s' already exists\n", dbName)
	} else {
		// Create database
		_, err = pool.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %q", dbName))
		if err != nil {
			log.Fatalf("Error creating database: %v", err)
		}
		fmt.Printf("Database '%s' created successfully!\n", dbName)
	}
}
