package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	repo "kinetix-api/internal/adapters/postgresql/sqlc"
)

// fakeDB implements TxStarter and hands out recorded fake transactions.
type fakeDB struct {
	tx          *fakeTx
	opts        pgx.TxOptions
	beginN      int
	rollbackErr error
}

func (f *fakeDB) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	f.beginN++
	f.opts = opts
	tx := newFakeTx()
	if f.rollbackErr != nil {
		tx.rollbackErr = f.rollbackErr
	}
	f.tx = tx
	return tx, nil
}

func newFakeTx() *fakeTx { return &fakeTx{id: uuid.New()} }

// fakeTx implements pgx.Tx recording commit/rollback decisions.
type fakeTx struct {
	id          uuid.UUID
	rollbackErr error
	committed   bool
	rolledback  bool
}

func (t *fakeTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, errors.New("nested tx not supported")
}
func (t *fakeTx) Commit(ctx context.Context) error { t.committed = true; return nil }
func (t *fakeTx) Rollback(ctx context.Context) error {
	t.rolledback = true
	return t.rollbackErr
}
func (t *fakeTx) Conn() *pgx.Conn { panic("unused") }
func (t *fakeTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	panic("unused")
}
func (t *fakeTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { panic("unused") }
func (t *fakeTx) LargeObjects() pgx.LargeObjects                               { panic("unused") }
func (t *fakeTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	panic("unused")
}
func (t *fakeTx) QueryFunc(ctx context.Context, sql string, args []any, scans []any, f func(args []any) error) (pgconn.CommandTag, error) {
	panic("unused")
}
func (t *fakeTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	panic("unused")
}
func (t *fakeTx) Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error) {
	panic("unused")
}
func (t *fakeTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { panic("unused") }

func TestExecTx_CommitsOnSuccess(t *testing.T) {
	db := &fakeDB{}
	s := Store{db: db, Queries: repo.New(db.tx)} // Queries unused by ExecTx

	err := s.ExecTx(context.Background(), func(q *repo.Queries) error {
		if q == nil {
			t.Fatal("fn must receive tx-scoped queries")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ExecTx: %v", err)
	}
	if !db.tx.committed || db.tx.rolledback {
		t.Errorf("commit=%v rollback=%v, want committed only", db.tx.committed, db.tx.rolledback)
	}
	if db.beginN != 1 {
		t.Errorf("BeginTx calls = %d", db.beginN)
	}
}

func TestExecTx_RollsBackOnError(t *testing.T) {
	db := &fakeDB{}
	s := Store{db: db}
	sentinel := errors.New("user insert failed")

	err := s.ExecTx(context.Background(), func(q *repo.Queries) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want the fn error", err)
	}
	if !db.tx.rolledback || db.tx.committed {
		t.Errorf("rollback=%v commit=%v, want rolled back only", db.tx.rolledback, db.tx.committed)
	}
}

func TestExecTx_ReportsRollbackFailure(t *testing.T) {
	db := &fakeDB{rollbackErr: errors.New("connection lost")}
	s := Store{db: db}

	err := s.ExecTx(context.Background(), func(q *repo.Queries) error { return errors.New("boom") })
	if err == nil || !errors.Is(err, db.tx.rollbackErr) {
		t.Fatalf("err = %v, want rollback error included", err)
	}
	if db.tx.committed {
		t.Error("commit must not run after failures")
	}
}
