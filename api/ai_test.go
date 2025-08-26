package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/garnizeh/rag/api"
	"github.com/garnizeh/rag/internal/models"
	"github.com/gorilla/mux"
)

// minimal fake repo implementing only ContextRepo methods used by the handler
type fakeContextRepo struct {
	history map[int64]map[int64]string // engineerID -> historyID -> contextJSON
}

func (f *fakeContextRepo) UpsertEngineerContext(ctx context.Context, engineerID int64, contextJSON string, appliedBy string) (int64, error) {
	// increment version like behaviour: use len(hist)+1
	if f.history == nil {
		f.history = map[int64]map[int64]string{}
	}
	if f.history[engineerID] == nil {
		f.history[engineerID] = map[int64]string{}
	}
	newID := int64(len(f.history[engineerID]) + 1)
	f.history[engineerID][newID] = contextJSON
	return newID, nil
}

func (f *fakeContextRepo) GetEngineerContext(ctx context.Context, engineerID int64) (string, int64, error) {
	return "", 0, nil
}

func (f *fakeContextRepo) CreateContextHistory(ctx context.Context, engineerID int64, contextJSON string, changesJSON *string, conflictsJSON *string, appliedBy string, version int64) (int64, error) {
	return 0, nil
}

func (f *fakeContextRepo) ListContextHistory(ctx context.Context, engineerID int64) ([]models.ContextHistory, error) {
	var out []models.ContextHistory
	if f.history == nil {
		return out, nil
	}
	m := f.history[engineerID]
	for id, c := range m {
		out = append(out, models.ContextHistory{ID: id, EngineerID: engineerID, ContextJSON: c})
	}
	return out, nil
}

// implement the new method
func (f *fakeContextRepo) GetContextHistoryByID(ctx context.Context, engineerID int64, historyID int64) (*models.ContextHistory, error) {
	if f.history == nil {
		return nil, nil
	}
	m := f.history[engineerID]
	if m == nil {
		return nil, nil
	}
	c, ok := m[historyID]
	if !ok {
		return nil, nil
	}
	return &models.ContextHistory{ID: historyID, EngineerID: engineerID, ContextJSON: c}, nil
}

// repositoryContextHistory mirrors the subset of models.ContextHistory used by the handler
// repositoryContextHistory not needed; using internal models.ContextHistory instead

// Because the handler expects repository.ContextRepo, but our fake only implements the subset
// we need to create a small adapter that satisfies the interface methods used by the handler.
// For tests we will construct AIHandler with schema and template repos nil and our fake as contextRepo.

func TestRollbackHandler_NotFound(t *testing.T) {
	h := api.NewAIHandler(nil, nil, nil, &fakeContextRepo{})
	req := httptest.NewRequest("POST", "/v1/ai/context/rollback/123?history_id=1", nil)
	// route variables are provided by mux; set them manually in context by using URL with pattern
	req = muxSetVars(req, map[string]string{"engineer_id": "123"})
	w := httptest.NewRecorder()

	h.RollbackContextHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing history, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRollbackHandler_Success(t *testing.T) {
	f := &fakeContextRepo{history: map[int64]map[int64]string{123: {1: "{\"name\":\"old\"}"}}}
	h := api.NewAIHandler(nil, nil, nil, f)
	req := httptest.NewRequest("POST", "/v1/ai/context/rollback/123?history_id=1", nil)
	req = muxSetVars(req, map[string]string{"engineer_id": "123"})
	w := httptest.NewRecorder()

	h.RollbackContextHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 ok, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "applied_version") {
		t.Fatalf("expected applied_version in response, got body=%s", w.Body.String())
	}
}

func muxSetVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}
