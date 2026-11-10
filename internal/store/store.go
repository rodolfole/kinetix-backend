package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "kinetix-api/internal/adapters/postgresql/sqlc"
)

// TxStarter abstracts the concrete pool so ExecTx is unit-testable with a
// fake transaction (ADR-14).
type TxStarter interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

// Store wraps a pgxpool and the sqlc Queries bound to it.
type Store struct {
	db      TxStarter
	Queries *repo.Queries
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		db:      db,
		Queries: repo.New(db),
	}
}

// WithQueries returns a Store clone bound to the given (usually
// transaction-scoped) Queries. Persist-transparent for adapters (ADR-14).
func (s *Store) WithQueries(q *repo.Queries) *Store {
	return &Store{db: s.db, Queries: q}
}

// ExecTx runs fn inside a database transaction with tx-scoped Queries.
// Multi-row writes (auth registration, registrations, webhook paid+payout)
// must use it: a failure rolls every write in fn back.
func (s *Store) ExecTx(ctx context.Context, fn func(q *repo.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if err := fn(repo.New(tx)); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx failed: %v (rollback failed: %w)", err, rbErr)
		}
		return err
	}
	return tx.Commit(ctx)
}
