package api_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/garnizeh/rag/api"
)

func TestSystemHandlers(t *testing.T) {
	h := &api.SystemHandler{}

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
	vh := h.VersionHandler("1.2.3", "2025-08-24T00:00:00Z")
	req2 := httptest.NewRequest(http.MethodGet, "/version", nil)
	w2 := httptest.NewRecorder()
	vh(w2, req2)
	res2 := w2.Result()
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("version: expected 200 got %d", res2.StatusCode)
	}
	b2, _ := io.ReadAll(res2.Body)
	if !strings.Contains(string(b2), `"version":"1.2.3"`) || !strings.Contains(string(b2), `"buildTime":"2025-08-24T00:00:00Z"`) {
		t.Fatalf("version: unexpected body %s", string(b2))
	}
}
