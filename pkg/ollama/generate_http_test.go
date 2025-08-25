package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/garnizeh/rag/internal/config"
	"github.com/garnizeh/rag/pkg/ollama"
)

// writeSequence writes each object as a JSON line and flushes; useful to simulate Ollama's streaming.
func writeSequence(w http.ResponseWriter, seq []map[string]any, delay time.Duration) {
	enc := json.NewEncoder(w)
	for i, obj := range seq {
		_ = enc.Encode(obj)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		if i < len(seq)-1 && delay > 0 {
			time.Sleep(delay)
		}
	}
}

func TestClient_Generate_Retries_Backoff_Succeeds(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			a := atomic.AddInt32(&attempts, 1)
			if a == 1 {
				// transient error
				http.Error(w, "temporary", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			writeSequence(w, []map[string]any{{"response": "ok", "done": true}}, 0)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := config.OllamaConfig{BaseURL: srv.URL, Timeout: 2 * time.Second, Retries: 2, Backoff: 10 * time.Millisecond, CircuitFailureThreshold: 10}
	client, err := ollama.NewClient(cfg, srv.Client())
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	ctx := context.Background()
	res, err := client.Generate(ctx, "m", "p")
	if err != nil {
		t.Fatalf("Generate expected success after retry, got error: %v", err)
	}
	if res.Meta == nil {
		t.Fatalf("expected meta in result")
	}
	if _, ok := res.Meta["latency_ms"]; !ok {
		t.Fatalf("expected latency_ms in meta")
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestClient_CircuitBreaker_Opens(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			atomic.AddInt32(&attempts, 1)
			http.Error(w, "permanent", http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := config.OllamaConfig{BaseURL: srv.URL, Timeout: 1 * time.Second, Retries: 0, Backoff: 1 * time.Millisecond, CircuitFailureThreshold: 2, CircuitReset: 1 * time.Minute}
	client, err := ollama.NewClient(cfg, srv.Client())
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	ctx := context.Background()
	// first two calls should return an error (but not ErrCircuitOpen).
	for i := 0; i < 2; i++ {
		if _, err := client.Generate(ctx, "m", "p"); err == nil {
			t.Fatalf("expected error on attempt %d", i+1)
		}
	}

	// next call should hit circuit open
	if _, err := client.Generate(ctx, "m", "p"); err != ollama.ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}
