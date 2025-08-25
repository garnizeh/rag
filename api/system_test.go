package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/garnizeh/rag/api"
	"github.com/garnizeh/rag/internal/db"
)

func TestSystemHandlers(t *testing.T) {
	// NewSystemHandler now accepts database and ai engine parameters; tests can
	// pass nil for these when only testing static handlers.
	h := api.NewSystemHandler("1.2.3", "2025-08-24T00:00:00Z", nil, nil)

	// HealthHandler
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.HealthHandler(w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("health: expected 200 got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {

		t.Fatalf("health: expected json content-type, got %q", ct)
	}
	b, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(b), `"status":"ok"`) {
		t.Fatalf("health: unexpected body %s", string(b))
	}

	// VersionHandler
	req2 := httptest.NewRequest(http.MethodGet, "/version", nil)
	w2 := httptest.NewRecorder()
	h.VersionHandler(w2, req2)
	res2 := w2.Result()
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("version: expected 200 got %d", res2.StatusCode)
	}
	b2, _ := io.ReadAll(res2.Body)
	if !strings.Contains(string(b2), `"version":"1.2.3"`) || !strings.Contains(string(b2), `"buildTime":"2025-08-24T00:00:00Z"`) {
		t.Fatalf("version: unexpected body %s", string(b2))
	}

	// ReadinessHandler: when DB is nil the handler should report unready (503)
	req3 := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w3 := httptest.NewRecorder()
	h.ReadinessHandler(w3, req3)
	res3 := w3.Result()
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("ready(nil db): expected 503 got %d", res3.StatusCode)
	}
	b3, _ := io.ReadAll(res3.Body)
	if !strings.Contains(string(b3), `"status":"unready"`) {
		t.Fatalf("ready(nil db): unexpected body %s", string(b3))
	}

	// When DB is reachable and AI engine is nil, readiness should be OK and AI marked degraded
	database, err := db.New(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer database.Close()

	h2 := api.NewSystemHandler("1.2.3", "2025-08-24T00:00:00Z", database, nil)
	req4 := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w4 := httptest.NewRecorder()
	h2.ReadinessHandler(w4, req4)
	res4 := w4.Result()
	defer res4.Body.Close()
	if res4.StatusCode != http.StatusOK {
		t.Fatalf("ready(db ok): expected 200 got %d", res4.StatusCode)
	}
	b4, _ := io.ReadAll(res4.Body)
	if !strings.Contains(string(b4), `"status":"ready"`) || !strings.Contains(string(b4), `"ai":"degraded"`) || !strings.Contains(string(b4), `"db":"ok"`) {
		t.Fatalf("ready(db ok): unexpected body %s", string(b4))
	}

	// LiveHandler should report process liveness
	req5 := httptest.NewRequest(http.MethodGet, "/live", nil)
	w5 := httptest.NewRecorder()
	h2.LiveHandler(w5, req5)
	res5 := w5.Result()
	defer res5.Body.Close()
	if res5.StatusCode != http.StatusOK {
		t.Fatalf("live: expected 200 got %d", res5.StatusCode)
	}
}
