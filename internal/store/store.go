package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "kinetix-api/internal/adapters/postgresql/sqlc"
)

// Store define la estructura que orquestará la DB y las Queries
type Store struct {
	db      *pgxpool.Pool
	Queries *repo.Queries
}

// NewStore crea una nueva instancia del Store
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		db:      db,
		Queries: repo.New(db),
	}
}

// WithTransaction ejecuta una función dentro de una transacción de base de datos
func (s *Store) WithTransaction(ctx context.Context, fn func(*repo.Queries) error) error {
	// 1. Iniciar la transacción
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}

	// 2. Asegurar rollback si hay error
	defer func() {
		if tx != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// 3. Crear un objeto Queries vinculado a esta transacción específica
	q := s.Queries.WithTx(tx)

	// 4. Ejecutar la función callback que contiene la lógica del Service
	err = fn(q)
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	// 5. Si todo salió bien, hacemos Commit
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	// Prevenir rollback defer después de commit exitoso
	tx = nil

	return nil
}

// GetPool devuelve el pool de conexiones por si se necesita acceso directo de bajo nivel
func (s *Store) GetPool() *pgxpool.Pool {
	return s.db
}
