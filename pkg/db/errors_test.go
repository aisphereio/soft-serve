package db

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
)

type codedSQLiteError struct{ code int }

func (e codedSQLiteError) Error() string { return "sqlite failure" }
func (e codedSQLiteError) Code() int     { return e.code }

func TestWrapErrorBadNoRows(t *testing.T) {
	for _, e := range []error{
		fmt.Errorf("foo"),
		errors.New("bar"),
		codedSQLiteError{code: 1},
	} {
		if err := WrapError(e); err != e {
			t.Errorf("WrapError(%v) => %v, want %v", e, err, e)
		}
	}
}

func TestWrapErrorGoodNoRows(t *testing.T) {
	if err := WrapError(sql.ErrNoRows); err != ErrRecordNotFound {
		t.Errorf("WrapError(sql.ErrNoRows) => %v, want %v", err, ErrRecordNotFound)
	}
}

func TestWrapErrorSQLiteConstraintWithoutDriverDependency(t *testing.T) {
	for _, code := range []int{19, 1555, 787, 2067} {
		if err := WrapError(codedSQLiteError{code: code}); err != ErrDuplicateKey {
			t.Errorf("WrapError(sqlite code %d) => %v, want %v", code, err, ErrDuplicateKey)
		}
	}
}
