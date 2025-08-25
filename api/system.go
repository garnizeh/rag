package api

import (
	"context"
	"net/http"
	"time"

	"github.com/garnizeh/rag/internal/ai"
	"github.com/garnizeh/rag/internal/db"
)

type SystemHandler struct {
	version   string
	buildTime string
	db        *db.DB
	ai        *ai.Engine
}

func NewSystemHandler(version, buildTime string, database *db.DB, engine *ai.Engine) *SystemHandler {
	return &SystemHandler{version: version, buildTime: buildTime, db: database, ai: engine}
}

func (h *SystemHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"status":  "ok",
		"service": "rag",
	}
	writeJSON(w, resp, http.StatusOK)
}

// LiveHandler returns a simple liveness response (process is up).
func (h *SystemHandler) LiveHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"status":  "alive",
		"service": "rag",
	}
	writeJSON(w, resp, http.StatusOK)
}

func (h *SystemHandler) VersionHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"version":   h.version,
		"buildTime": h.buildTime,
	}
	writeJSON(w, resp, http.StatusOK)
}

// ReadinessHandler returns readiness of the application considering DB and AI availability.
// DB must be reachable for the service to be considered ready. AI being unavailable marks
// the service as degraded but does not make it unready.
func (h *SystemHandler) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	status := "ready"
	details := map[string]string{}

	// Check DB connectivity with a short timeout
	if h.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := h.db.GetConn().PingContext(ctx); err != nil {
			details["db"] = "down"
			status = "unready"
		} else {
			details["db"] = "ok"
		}
	} else {
		details["db"] = "unknown"
		status = "unready"
	}

	if h.ai == nil {
		details["ai"] = "degraded"
		// AI degraded does not change overall readiness; keep status as-is
	} else {
		details["ai"] = "ok"
	}

	resp := map[string]any{
		"status":  status,
		"service": "rag",
		"details": details,
	}

	if status == "ready" {
		writeJSON(w, resp, http.StatusOK)
	} else {
		writeJSON(w, resp, http.StatusServiceUnavailable)
	}
}
