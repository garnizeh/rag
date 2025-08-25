package ollama_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/garnizeh/rag/internal/config"
	"github.com/garnizeh/rag/pkg/ollama"
)

func TestClient_ListModelsAndHealth_Success(t *testing.T) {
	// mock server that returns a simple models list
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/models" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"name":"test-model"}]`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := config.OllamaConfig{BaseURL: srv.URL, Timeout: 2 * time.Second, Retries: 0}
	client, err := ollama.NewClient(cfg, srv.Client())
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	ctx := context.Background()
	// ListModels should return the model from the fake server
	models, err := client.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}
	if len(models) != 1 || models[0].Name != "test-model" {
		t.Fatalf("unexpected models: %#v", models)
	}

	// Health should succeed because ListModels returns at least one
	if err := client.Health(ctx); err != nil {
		t.Fatalf("Health failed: %v", err)
	}
}

func TestClient_Health_NoModels_Fails(t *testing.T) {
	// server returns empty array for models
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/models" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := config.OllamaConfig{BaseURL: srv.URL, Timeout: 2 * time.Second, Retries: 0}
	client, err := ollama.NewClient(cfg, srv.Client())
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	ctx := context.Background()
	if err := client.Health(ctx); err == nil {
		t.Fatalf("expected Health to fail when no models returned")
	}
}

func TestClient_Generate_Streaming_Success(t *testing.T) {
	// server will stream two JSON objects; client should capture the last one
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			// send first chunk
			if _, err := w.Write([]byte("{" + "\"response\":\"one\",\"done\":false}" + "\n")); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(10 * time.Millisecond)
			// send final chunk
			_, _ = w.Write([]byte("{" + "\"response\":\"final\",\"done\":true}" + "\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := config.OllamaConfig{BaseURL: srv.URL, Timeout: 2 * time.Second, Retries: 0}
	client, err := ollama.NewClient(cfg, srv.Client())
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	ctx := context.Background()
	res, err := client.Generate(ctx, "test-model", "prompt")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if res.Text == "" || !contains(res.Text, "final") {
		t.Fatalf("unexpected Generate.Text: %q", res.Text)
	}
}

func TestClient_Generate_Non200_Fails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := config.OllamaConfig{BaseURL: srv.URL, Timeout: 2 * time.Second, Retries: 0}
	client, err := ollama.NewClient(cfg, srv.Client())
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	ctx := context.Background()
	if _, err := client.Generate(ctx, "test-model", "prompt"); err == nil {
		t.Fatalf("expected Generate to fail on non-200")
	}
}

func TestClient_Generate_MalformedJSON_Fails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			// send invalid JSON
			_, _ = w.Write([]byte(`{ this is : not json `))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := config.OllamaConfig{BaseURL: srv.URL, Timeout: 2 * time.Second, Retries: 0}
	client, err := ollama.NewClient(cfg, srv.Client())
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	ctx := context.Background()
	if _, err := client.Generate(ctx, "test-model", "prompt"); err == nil {
		t.Fatalf("expected Generate to fail on malformed JSON")
	}
}

// contains checks substring without importing strings repeatedly in tests
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && (indexOf(s, sub) >= 0)))
}

// indexOf is a tiny fallback to avoid adding another import; it's simple and sufficient for tests
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
