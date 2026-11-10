package store

import (
	"github.com/jackc/pgx/v5/pgxpool"

	repo "kinetix-api/internal/adapters/postgresql/sqlc"
)

// Store wraps a pgxpool and the sqlc Queries bound to it.
type Store struct {
	db      *pgxpool.Pool
	Queries *repo.Queries
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		db:      db,
		Queries: repo.New(db),
	}
}
