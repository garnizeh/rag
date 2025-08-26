package ai

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMergeAIResponse_EmptyExisting(t *testing.T) {
	resp := &AIResponse{
		Summary: "Worked on project X",
		Entities: struct {
			People       []string `json:"people"`
			Projects     []string `json:"projects"`
			Technologies []string `json:"technologies"`
		}{
			People:       []string{"Alice"},
			Projects:     []string{"Project X"},
			Technologies: []string{"Go"},
		},
		ContextUpdate: true,
	}

	res, err := MergeAIResponse(context.Background(), nil, resp)
	if err != nil {
		t.Fatalf("merge error: %v", err)
	}

	if len(res.Changes) == 0 {
		t.Fatalf("expected changes, got none")
	}

	// verify keys exist
	if _, ok := res.Merged["people"]; !ok {
		t.Fatalf("people not set")
	}
	if _, ok := res.Merged["projects"]; !ok {
		t.Fatalf("projects not set")
	}
	if _, ok := res.Merged["technologies"]; !ok {
		t.Fatalf("technologies not set")
	}
	if _, ok := res.Merged["summary"]; !ok {
		t.Fatalf("summary not set")
	}
}

func TestMergeAIResponse_InvalidNames(t *testing.T) {
	resp := &AIResponse{
		Entities: struct {
			People       []string `json:"people"`
			Projects     []string `json:"projects"`
			Technologies []string `json:"technologies"`
		}{
			People:       []string{"   ", string(make([]byte, 260))},
			Projects:     []string{"GoodProj"},
			Technologies: []string{},
		},
	}

	_, err := MergeAIResponse(context.Background(), nil, resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiffContexts_Basic(t *testing.T) {
	before := map[string]any{"a": "1", "b": "2"}
	after := map[string]any{"a": "1", "b": "3", "c": "new"}

	bbs, _ := json.Marshal(before)
	abs, _ := json.Marshal(after)

	diffs, err := DiffContexts(bbs, abs)
	if err != nil {
		t.Fatalf("diff error: %v", err)
	}

	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}
}
