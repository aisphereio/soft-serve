package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"charm.land/log/v2"
	"github.com/aisphereio/soft-serve/pkg/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"  // postgres driver
	_ "modernc.org/sqlite" // sqlite driver
)

// DB is the interface for a Soft Serve database.
type DB struct {
	*sqlx.DB
	logger *log.Logger
	ownsDB bool
}

// Open opens and owns a database connection pool.
func Open(ctx context.Context, driverName string, dsn string) (*DB, error) {
	database, err := sqlx.ConnectContext(ctx, driverName, dsn)
	if err != nil {
		return nil, err
	}

	return newDB(ctx, database, true), nil
}

// NewWithSQLDB wraps an existing database/sql pool for use by Soft Serve.
// When ownsDB is false, Close leaves the supplied pool open so the embedding
// application remains responsible for its lifecycle.
func NewWithSQLDB(ctx context.Context, sqlDB *sql.DB, driverName string, ownsDB bool) (*DB, error) {
	if sqlDB == nil {
		return nil, errors.New("soft-serve/db: SQL database is required")
	}
	if driverName == "" {
		return nil, errors.New("soft-serve/db: driver name is required")
	}

	return newDB(ctx, sqlx.NewDb(sqlDB, driverName), ownsDB), nil
}

func newDB(ctx context.Context, database *sqlx.DB, ownsDB bool) *DB {
	d := &DB{
		DB:     database,
		ownsDB: ownsDB,
	}
	if config.IsVerbose() {
		d.logger = log.FromContext(ctx).WithPrefix("db")
	}
	return d
}

// Close implements db.DB.
func (d *DB) Close() error {
	if d == nil || d.DB == nil || !d.ownsDB {
		return nil
	}
	return d.DB.Close()
}

// Tx is a database transaction.
type Tx struct {
	*sqlx.Tx
	logger *log.Logger
}

// Transaction implements db.DB.
func (d *DB) Transaction(fn func(tx *Tx) error) error {
	return d.TransactionContext(context.Background(), fn)
}

// TransactionContext implements db.DB.
func (d *DB) TransactionContext(ctx context.Context, fn func(tx *Tx) error) error {
	txx, err := d.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	tx := &Tx{txx, d.logger}
	if err := fn(tx); err != nil {
		return rollback(tx, err)
	}

	if err := tx.Commit(); err != nil {
		if errors.Is(err, sql.ErrTxDone) {
			// this is ok because whoever did finish the tx should have also written the error already.
			return nil
		}
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func rollback(tx *Tx, err error) error {
	if rerr := tx.Rollback(); rerr != nil {
		if errors.Is(rerr, sql.ErrTxDone) {
			return err
		}
		return fmt.Errorf("failed to rollback: %s: %w", err.Error(), rerr)
	}
	return err
}
