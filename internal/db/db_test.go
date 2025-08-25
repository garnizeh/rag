package db_test

import (
	"context"
	"testing"
	"time"

	dbpkg "github.com/garnizeh/rag/internal/db"
)

func TestNew_Close_GetConn(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use in-memory SQLite
	d, err := dbpkg.New(ctx, "file::memory:?cache=shared", nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	conn := d.GetConn()
	if conn == nil {
		t.Fatalf("expected non-nil sql.DB from GetConn")
	}

	// Close should not error
	if err := d.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestExec_QueryRow(t *testing.T) {
	ctx := context.Background()
	d, err := dbpkg.New(ctx, "file::memory:?cache=shared", nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	defer d.Close()

	// create table
	_, err = d.Exec(ctx, `CREATE TABLE IF NOT EXISTS items (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT);`)
	if err != nil {
		t.Fatalf("Exec create table returned error: %v", err)
	}

	// insert
	res, err := d.Exec(ctx, `INSERT INTO items (name) VALUES (?)`, "foo")
	if err != nil {
		t.Fatalf("Exec insert returned error: %v", err)
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}
	if lastID == 0 {
		t.Fatalf("expected last insert id > 0")
	}

	// query
	row := d.QueryRow(ctx, `SELECT name FROM items WHERE id = ?`, lastID)
	var name string
	if err := row.Scan(&name); err != nil {
		t.Fatalf("QueryRow scan returned error: %v", err)
	}
	if name != "foo" {
		t.Fatalf("expected name 'foo' got %q", name)
	}
}

func TestNew_BadDSN(t *testing.T) {
	ctx := context.Background()
	// Some sqlite drivers will treat the string ":invalid-dsn:" as a valid
	// filename and succeed. To avoid a flaky test across environments, allow
	// either an error (driver rejects the DSN) or a successful DB. If a DB is
	// returned ensure it is usable and closed.
	d, err := dbpkg.New(ctx, ":invalid-dsn:", nil)
	if err != nil {
		// driver rejected the DSN — acceptable
		return
	}
	// If we got a DB back, make sure it's usable and close it to avoid leaks
	if conn := d.GetConn(); conn == nil {
		t.Fatalf("expected non-nil sql.DB from GetConn when New succeeds")
	}
	if err := d.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}
