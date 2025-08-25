package db

import (
	"context"
	"database/sql"
	"fmt"

	"os"

	"log/slog"

	_ "modernc.org/sqlite"
)

// DB wraps the sql.DB for connection management
type DB struct {
	conn   *sql.DB
	logger *slog.Logger
}

// New creates a new DB connection
func New(ctx context.Context, dsn string, logger *slog.Logger) (*DB, error) {
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return &DB{conn: conn, logger: logger}, nil
}

// Close closes the DB connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Exec executes a query
func (db *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.conn.ExecContext(ctx, query, args...)
}

// QueryRow executes a query that is expected to return at most one row
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return db.conn.QueryRowContext(ctx, query, args...)
}

// QueryRows executes a query that returns multiple rows
func (db *DB) QueryRows(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.conn.QueryContext(ctx, query, args...)
}

// GetConn returns the underlying sql.DB
func (db *DB) GetConn() *sql.DB {
	return db.conn
}
