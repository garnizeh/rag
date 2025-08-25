package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/garnizeh/rag/api"
	"github.com/garnizeh/rag/internal/db"
	sqlite "github.com/garnizeh/rag/internal/repository/sqlite"
)

func setupServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	ctx := context.Background()
	d, err := db.New(ctx, "file::memory:?cache=shared", nil)
	if err != nil {
		t.Fatalf("db.New: %v", err)
	}

	// create minimal schema
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS engineers (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, email TEXT, updated INTEGER, password_hash TEXT);`,
		`CREATE TABLE IF NOT EXISTS raw_activities (id INTEGER PRIMARY KEY AUTOINCREMENT, engineer_id INTEGER, activity TEXT, created INTEGER);`,
		`CREATE TABLE IF NOT EXISTS processing_jobs (id INTEGER PRIMARY KEY AUTOINCREMENT, status TEXT, created INTEGER);`,
	}
	for _, s := range stmts {
		if _, err := d.Exec(ctx, s); err != nil {
			d.Close()
			t.Fatalf("setup schema: %v", err)
		}
	}

	repo := sqlite.New(d, nil)
	ah := api.NewActivitiesHandler(repo, repo)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/activities", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			ah.CreateActivity(w, r)
		case http.MethodGet:
			ah.ListActivities(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	srv := httptest.NewServer(mux)
	return srv, func() { srv.Close(); d.Close() }
}

func TestCreateAndListActivities(t *testing.T) {
	srv, cleanup := setupServer(t)
	defer cleanup()

	// create 3 activities
	for range 3 {
		payload := map[string]any{"engineer_id": 1, "activity": "act"}
		b, _ := json.Marshal(payload)
		res, err := http.Post(srv.URL+"/v1/activities", "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatalf("post request failed: %v", err)
		}
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 created got %d", res.StatusCode)
		}
	}

	// page1
	res1, err := http.Get(srv.URL + "/v1/activities?engineer_id=1&limit=2&offset=0")
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if res1.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", res1.StatusCode)
	}
	var body1 map[string]any
	if err := json.NewDecoder(res1.Body).Decode(&body1); err != nil {
		t.Fatalf("decode page1: %v", err)
	}
	if int(body1["total"].(float64)) != 3 {
		t.Fatalf("expected total 3 got %v", body1["total"])
	}
	items1 := body1["items"].([]any)
	if len(items1) != 2 {
		t.Fatalf("expected 2 items on page1 got %d", len(items1))
	}

	// page2
	res2, err := http.Get(srv.URL + "/v1/activities?engineer_id=1&limit=2&offset=2")
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", res2.StatusCode)
	}
	var body2 map[string]any
	if err := json.NewDecoder(res2.Body).Decode(&body2); err != nil {
		t.Fatalf("decode page2: %v", err)
	}
	if int(body2["total"].(float64)) != 3 {
		t.Fatalf("expected total 3 got %v", body2["total"])
	}
	items2 := body2["items"].([]any)
	if len(items2) != 1 {
		t.Fatalf("expected 1 item on page2 got %d", len(items2))
	}

	// ensure no duplicate IDs across pages
	seen := map[float64]bool{}
	for _, it := range items1 {
		m := it.(map[string]any)
		seen[m["id"].(float64)] = true
	}
	for _, it := range items2 {
		m := it.(map[string]any)
		id := m["id"].(float64)
		if seen[id] {
			t.Fatalf("duplicate id across pages: %v", id)
		}
	}
}
