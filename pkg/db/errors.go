package db

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

const sqliteConstraintBaseCode = 19

type sqliteErrorCoder interface {
	Code() int
}

var (
	// ErrDuplicateKey is a constraint violation error.
	ErrDuplicateKey = errors.New("duplicate key value violates table constraint")

	// ErrRecordNotFound is returned when a record is not found.
	ErrRecordNotFound = sql.ErrNoRows
)

// WrapError unifies driver-specific database errors without importing a
// concrete SQLite implementation. This keeps the reusable db package safe for
// embedding applications that register a different SQLite driver.
func WrapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRecordNotFound
	}

	// SQLite extended result codes preserve the base code in the low byte.
	// SQLITE_CONSTRAINT and all of its PRIMARYKEY/FOREIGNKEY/UNIQUE variants
	// therefore match without linking modernc.org/sqlite into this package.
	var sqliteErr sqliteErrorCoder
	if errors.As(err, &sqliteErr) && sqliteErr.Code()&0xff == sqliteConstraintBaseCode {
		return ErrDuplicateKey
	}

	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" || pgErr.Code == "23503" || pgErr.Code == "23514" {
			return ErrDuplicateKey
		}
	}
	return err
}
