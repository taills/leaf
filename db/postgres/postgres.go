// Package postgres provides a PostgreSQL-backed db.Store implementation for
// Leaf, built on the jackc/pgx/v5 driver (PostgreSQL 18 compatible) through its
// database/sql-compatible stdlib adapter.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/taills/leaf/v2/db"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
)

// uniqueViolation is the PostgreSQL SQLSTATE code for a unique-constraint
// violation. See https://www.postgresql.org/docs/current/errcodes-appendix.html
const uniqueViolation = "23505"

// Open connects to PostgreSQL using the given DSN (libpq URL or key/value DSN,
// e.g. "postgres://user:pass@localhost:5432/game?sslmode=disable") and returns
// a ready-to-use db.Store.
func Open(dsn string) (db.Store, error) {
	return OpenWithTimeout(dsn, 10*time.Second)
}

// OpenWithTimeout behaves like Open but bounds the connection and initial
// schema setup with a timeout.
func OpenWithTimeout(dsn string, timeout time.Duration) (db.Store, error) {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db.OpenSQL(ctx, sqlDB, db.DialectPostgres, isDup)
}

// isDup reports whether err is a PostgreSQL unique-constraint violation.
func isDup(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == uniqueViolation
	}
	return false
}
