package ollama_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/garnizeh/rag/pkg/ollama"
)

func TestGenerateResult_MarshalMeta(t *testing.T) {
	gr := ollama.GenerateResult{Text: "ok", Raw: json.RawMessage(`{"x":1}`), Meta: map[string]any{"model": "m", "latency_ms": 123}}
	b, err := json.Marshal(gr)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	s := string(b)
	if s == "" || !strings.Contains(s, "latency_ms") || !strings.Contains(s, "model") {
		t.Fatalf("unexpected marshaled result: %s", s)
	}
}
