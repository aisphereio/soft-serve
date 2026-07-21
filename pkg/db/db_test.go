package db

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestOpenUnknownDriver(t *testing.T) {
	_, err := Open(context.TODO(), "invalid", "")
	if err == nil {
		t.Error("Open(invalid) => nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown driver") {
		t.Errorf("Open(invalid) => %v, want error containing 'unknown driver'", err)
	}
}

func TestNewWithSQLDBDoesNotCloseNonOwnedPool(t *testing.T) {
	raw, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close() //nolint:errcheck

	wrapped, err := NewWithSQLDB(context.Background(), raw, "sqlite", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := wrapped.Close(); err != nil {
		t.Fatal(err)
	}
	if err := raw.PingContext(context.Background()); err != nil {
		t.Fatalf("non-owned pool was closed: %v", err)
	}
}

func TestNewWithSQLDBRejectsInvalidInput(t *testing.T) {
	if _, err := NewWithSQLDB(context.Background(), nil, "postgres", false); err == nil {
		t.Fatal("NewWithSQLDB(nil) => nil error, want error")
	}

	raw, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close() //nolint:errcheck
	if _, err := NewWithSQLDB(context.Background(), raw, "", false); err == nil {
		t.Fatal("NewWithSQLDB(empty driver) => nil error, want error")
	}
}
