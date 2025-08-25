package ollama

import (
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/garnizeh/rag/internal/config"
)

type testTransport struct{ called int32 }

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) { panic("not used") }
func (t *testTransport) CloseIdleConnections()                               { atomic.StoreInt32(&t.called, 1) }

func TestClient_Close_IdempotentAndCallsTransport(t *testing.T) {
	tr := &testTransport{}
	client := &http.Client{Transport: tr}
	cfg := config.OllamaConfig{BaseURL: "http://localhost:11434", Timeout: 1}
	c, err := NewClient(cfg, client)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}
	if atomic.LoadInt32(&tr.called) != 1 {
		t.Fatalf("expected CloseIdleConnections called once")
	}

	// second call should be a no-op
	if err := c.Close(); err != nil {
		t.Fatalf("Close second call error: %v", err)
	}
}
