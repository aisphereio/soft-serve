// Package db provides the database primitives used by Soft Serve.
//
// Standalone executables are responsible for registering their selected SQL
// drivers before calling Open. Embedding applications may instead supply an
// existing database/sql pool through NewWithSQLDB and keep ownership of that
// pool by passing ownsDB=false.
package db
