package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds database configuration
type Config struct {
	Host     string
	Port     string
	DBName   string
	User     string
	Password string
	SSLMode  string
}

// NewConnectionPool creates a new database connection pool using pgx/v5
func NewConnectionPool(cfg Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
		cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("error parsing database config: %w", err)
	}

	// Set connection pool settings
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	// Create the connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating connection pool: %w", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	log.Println("Successfully connected to database (pgx/v5)")
	return pool, nil
}

// ClosePool closes the database connection pool
func ClosePool(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
		log.Println("Database connection pool closed")
	}
}
