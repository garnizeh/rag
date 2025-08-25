package ollama

import (
	"net/http"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/garnizeh/rag/internal/config"
)

// TestClient_NoGoroutineLeak creates and closes many clients to detect obvious goroutine leaks.
// This is a best-effort smoke test; it checks that the number of goroutines doesn't grow
// significantly after creating and closing clients repeatedly.
func TestClient_NoGoroutineLeak(t *testing.T) {
	runtime.GC()
	before := runtime.NumGoroutine()

	var wg sync.WaitGroup
	n := 50
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{}
			cfg := config.OllamaConfig{BaseURL: "http://localhost:11434", Timeout: 1}
			c, err := NewClient(cfg, client)
			if err != nil {
				t.Errorf("new client: %v", err)
				return
			}
			if err := c.Close(); err != nil {
				t.Errorf("close: %v", err)
			}
		}()
	}
	wg.Wait()

	// give a little time for goroutines to exit
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	after := runtime.NumGoroutine()

	if after-before > 10 {
		t.Fatalf("possible goroutine leak: before=%d after=%d", before, after)
	}
}
