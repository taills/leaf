// Package db defines a database-agnostic Store abstraction for the Leaf
// framework. Concrete drivers live in sub-packages (db/sqlite, db/postgres)
// and all expose the same Store interface, so business code can switch the
// underlying database without any change.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// Dialect identifies the SQL flavor of the underlying driver. It is used to
// rewrite portable queries (written with "?" placeholders) into the syntax
// that the target database expects.
type Dialect int

const (
	// DialectSQLite targets SQLite (placeholders are "?").
	DialectSQLite Dialect = iota
	// DialectPostgres targets PostgreSQL (placeholders are "$1", "$2", ...).
	DialectPostgres
)

// Store is the database-agnostic interface that every Leaf database driver
// implements. The KV and sequence helpers cover the common game-server needs
// (auto-increment player ids, simple persistence), while Exec/Query provide a
// raw escape hatch for arbitrary SQL.
//
// Every method is goroutine safe.
type Store interface {
	// NextSeq atomically increments and returns the next value of the named
	// sequence, creating it (starting at 1) on first use.
	NextSeq(ctx context.Context, name string) (int64, error)

	// Get returns the value stored under key. ok reports whether the key
	// exists.
	Get(ctx context.Context, key string) (value []byte, ok bool, err error)
	// Set inserts or updates the value stored under key.
	Set(ctx context.Context, key string, value []byte) error
	// Del removes key. Removing a missing key is not an error.
	Del(ctx context.Context, key string) error

	// Exec runs a raw statement. Placeholders use "?" and are rewritten for
	// the active dialect.
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	// Query runs a raw query. Placeholders use "?" and are rewritten for the
	// active dialect.
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	// QueryRow runs a raw single-row query. Placeholders use "?" and are
	// rewritten for the active dialect.
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row

	// IsDup reports whether err is a unique-constraint (duplicate key)
	// violation.
	IsDup(err error) bool

	// DB exposes the underlying *sql.DB for advanced use.
	DB() *sql.DB
	// Close releases all resources held by the store.
	Close() error
}

// sqlStore is the shared database/sql-backed implementation. Both the SQLite
// and PostgreSQL drivers wrap it, differing only by dialect and the
// duplicate-key detection function they inject.
type sqlStore struct {
	db      *sql.DB
	dialect Dialect
	isDup   func(error) bool
}

// OpenSQL builds a Store on top of an already-open *sql.DB, provisioning the
// internal bookkeeping tables (_leaf_seq, _leaf_kv). Driver packages call this
// after registering and opening their database/sql driver.
func OpenSQL(ctx context.Context, sqlDB *sql.DB, dialect Dialect, isDup func(error) bool) (Store, error) {
	s := &sqlStore{db: sqlDB, dialect: dialect, isDup: isDup}
	if err := s.init(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return s, nil
}

func (s *sqlStore) blobType() string {
	if s.dialect == DialectPostgres {
		return "BYTEA"
	}
	return "BLOB"
}

func (s *sqlStore) init(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS _leaf_seq (name TEXT PRIMARY KEY, seq BIGINT NOT NULL)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS _leaf_kv (k TEXT PRIMARY KEY, v %s NOT NULL)`, s.blobType()),
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("leaf/db: init schema: %w", err)
		}
	}
	return nil
}

// rebind rewrites portable "?" placeholders into the dialect-specific form.
// SQLite keeps "?"; PostgreSQL needs positional "$1", "$2", ... placeholders.
func (s *sqlStore) rebind(query string) string {
	if s.dialect != DialectPostgres {
		return query
	}

	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
		} else {
			b.WriteByte(query[i])
		}
	}
	return b.String()
}

func (s *sqlStore) NextSeq(ctx context.Context, name string) (int64, error) {
	// Single atomic UPSERT-with-RETURNING; supported by SQLite (3.35+) and
	// PostgreSQL. On first insert seq=1; on conflict it is incremented.
	const q = `INSERT INTO _leaf_seq (name, seq) VALUES (?, 1) ` +
		`ON CONFLICT(name) DO UPDATE SET seq = _leaf_seq.seq + 1 RETURNING seq`
	var seq int64
	if err := s.db.QueryRowContext(ctx, s.rebind(q), name).Scan(&seq); err != nil {
		return 0, fmt.Errorf("leaf/db: NextSeq(%q): %w", name, err)
	}
	return seq, nil
}

func (s *sqlStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	const q = `SELECT v FROM _leaf_kv WHERE k = ?`
	var v []byte
	err := s.db.QueryRowContext(ctx, s.rebind(q), key).Scan(&v)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("leaf/db: Get(%q): %w", key, err)
	}
	return v, true, nil
}

func (s *sqlStore) Set(ctx context.Context, key string, value []byte) error {
	const q = `INSERT INTO _leaf_kv (k, v) VALUES (?, ?) ` +
		`ON CONFLICT(k) DO UPDATE SET v = excluded.v`
	if _, err := s.db.ExecContext(ctx, s.rebind(q), key, value); err != nil {
		return fmt.Errorf("leaf/db: Set(%q): %w", key, err)
	}
	return nil
}

func (s *sqlStore) Del(ctx context.Context, key string) error {
	const q = `DELETE FROM _leaf_kv WHERE k = ?`
	if _, err := s.db.ExecContext(ctx, s.rebind(q), key); err != nil {
		return fmt.Errorf("leaf/db: Del(%q): %w", key, err)
	}
	return nil
}

func (s *sqlStore) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, s.rebind(query), args...)
}

func (s *sqlStore) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, s.rebind(query), args...)
}

func (s *sqlStore) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, s.rebind(query), args...)
}

func (s *sqlStore) IsDup(err error) bool {
	if err == nil {
		return false
	}
	if s.isDup != nil {
		return s.isDup(err)
	}
	return false
}

func (s *sqlStore) DB() *sql.DB { return s.db }

func (s *sqlStore) Close() error { return s.db.Close() }
