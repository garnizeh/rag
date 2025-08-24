package api

import (
	"fmt"
	"net/http"
)

type SystemHandler struct{}

func (h *SystemHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status":"ok","service":"rag"}`)
}

func (h *SystemHandler) VersionHandler(version, buildTime string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"version":"%s","buildTime":"%s"}`, version, buildTime)
	}
}