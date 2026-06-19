// Package sqlite provides a SQLite-backed db.Store implementation for Leaf,
// built on the pure-Go (CGO-free) modernc.org/sqlite driver.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/taills/leaf/db"
	sqlite3 "modernc.org/sqlite"
	sqlite3lib "modernc.org/sqlite/lib"
)

// Open opens (or creates) a SQLite database at the given DSN and returns a
// ready-to-use db.Store. The DSN is a file path or any modernc.org/sqlite DSN,
// e.g. "game.db", "file:game.db?cache=shared", or ":memory:".
func Open(dsn string) (db.Store, error) {
	return OpenWithTimeout(dsn, 10*time.Second)
}

// OpenWithTimeout behaves like Open but bounds the initial schema setup with a
// timeout.
func OpenWithTimeout(dsn string, timeout time.Duration) (db.Store, error) {
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite allows only a single writer; serialize writes to avoid
	// "database is locked" errors under Leaf's concurrent access.
	sqlDB.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return db.OpenSQL(ctx, sqlDB, db.DialectSQLite, isDup)
}

// isDup reports whether err is a SQLite UNIQUE / PRIMARY KEY constraint
// violation.
func isDup(err error) bool {
	var e *sqlite3.Error
	if errors.As(err, &e) {
		code := e.Code()
		return code == sqlite3lib.SQLITE_CONSTRAINT_UNIQUE ||
			code == sqlite3lib.SQLITE_CONSTRAINT_PRIMARYKEY
	}
	return false
}
