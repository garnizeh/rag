package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB wraps the sql.DB for connection management
type DB struct {
	conn *sql.DB
}

// New creates a new DB connection
func New(ctx context.Context, dsn string) (*DB, error) {
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}
	return &DB{conn: conn}, nil
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

// GetConn returns the underlying sql.DB
func (db *DB) GetConn() *sql.DB {
	return db.conn
}
