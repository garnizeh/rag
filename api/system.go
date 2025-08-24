package api

import (
	"net/http"
)

type SystemHandler struct {
	version   string
	buildTime string
}

func NewSystemHandler(version, buildTime string) *SystemHandler {
	return &SystemHandler{version: version, buildTime: buildTime}
}

func (h *SystemHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"status":  "ok",
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
