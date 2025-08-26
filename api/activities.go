package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/garnizeh/rag/internal/models"
	"github.com/garnizeh/rag/pkg/repository"
)

type ActivitiesHandler struct {
	activityRepo repository.ActivityRepo
	jobRepo      repository.JobRepo
}

func NewActivitiesHandler(ar repository.ActivityRepo, jr repository.JobRepo) *ActivitiesHandler {
	return &ActivitiesHandler{activityRepo: ar, jobRepo: jr}
}

type postActivityRequest struct {
	EngineerID int64  `json:"engineer_id"` // TODO: get engineer ID from JWT
	Activity   string `json:"activity"`
	Timestamp  *int64 `json:"timestamp,omitempty"`
}

type postActivityResponse struct {
	ID int64 `json:"id"`
}

func (h *ActivitiesHandler) CreateActivity(w http.ResponseWriter, r *http.Request) {
	var req postActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// If engineer id not provided in body, try to read from context (set by JWT middleware)
	if req.EngineerID <= 0 {
		if v := r.Context().Value(CtxEngineerID); v == nil {
			if id, ok := v.(int64); ok {
				req.EngineerID = id
			}
		}
	}

	// Basic validation
	req.Activity = strings.TrimSpace(req.Activity)
	if req.EngineerID <= 0 || req.Activity == "" {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}
	if len(req.Activity) > 2000 {
		http.Error(w, "activity too long", http.StatusBadRequest)
		return
	}

	if req.Timestamp == nil || *req.Timestamp <= 0 {
		now := time.Now().UTC().UnixMicro()
		req.Timestamp = &now
	}
	// TODO: deduplication: simple in-memory check could be added; for now rely on consumer

	a := &models.Activity{EngineerID: req.EngineerID, Activity: req.Activity, Created: *req.Timestamp}
	id, err := h.activityRepo.CreateActivity(r.Context(), a)
	if err != nil {
		http.Error(w, "failed to store activity", http.StatusInternalServerError)
		return
	}

	// enqueue legacy processing job (short-term compatibility)
	job := &models.Job{Status: "pending"}
	if _, err := h.jobRepo.CreateJob(r.Context(), job); err != nil {
		// don't fail the request if job enqueue fails; log and continue
		fmt.Println("warning: failed to create job:", err)
	}

	// enqueue AI analysis job into the worker queue: ai.analyze_activity
	payloadObj := map[string]any{"engineer_id": req.EngineerID, "activity": req.Activity, "timestamp": *req.Timestamp}
	b, _ := json.Marshal(payloadObj)
	j := &models.BackgroundJob{Type: "ai.analyze_activity", Payload: b, Priority: 100, MaxAttempts: 3}
	if _, err := h.jobRepo.Enqueue(r.Context(), j); err != nil {
		fmt.Println("warning: failed to enqueue ai.analyze_activity job:", err)
	}

	writeJSON(w, postActivityResponse{ID: id}, http.StatusCreated)
}

func (h *ActivitiesHandler) ListActivities(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	engStr := q.Get("engineer_id")
	if engStr == "" {
		http.Error(w, "engineer_id is required", http.StatusBadRequest)
		return
	}
	engID, err := strconv.ParseInt(engStr, 10, 64)
	if err != nil || engID <= 0 {
		http.Error(w, "invalid engineer_id", http.StatusBadRequest)
		return
	}

	// pagination: limit and offset params
	limit := 50
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	acts, err := h.activityRepo.ListByEngineer(r.Context(), engID, limit, offset)
	if err != nil {
		http.Error(w, "failed to list activities", http.StatusInternalServerError)
		return
	}

	total, err := h.activityRepo.CountActivitiesByEngineer(r.Context(), engID)
	if err != nil {
		http.Error(w, "failed to count activities", http.StatusInternalServerError)
		return
	}

	if acts == nil {
		acts = []models.Activity{}
	}

	resp := map[string]any{
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"items":  acts,
	}

	writeJSON(w, resp, http.StatusOK)
}
