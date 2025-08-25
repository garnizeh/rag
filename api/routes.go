package api

import (
	"encoding/json"
	"net/http"

	"log/slog"

	"github.com/garnizeh/rag/internal/ai"
	"github.com/garnizeh/rag/internal/config"
	"github.com/garnizeh/rag/internal/db"
	"github.com/garnizeh/rag/pkg/repository"
	"github.com/gorilla/mux"
)

func SetupRoutes(
	cfg *config.Config,
	version string,
	buildTime string,
	repo repository.Repository,
	aiEngine *ai.Engine,
	database *db.DB,
	logger *slog.Logger,
) *mux.Router {
	r := mux.NewRouter()

	// Middleware chain
	r.Use(LoggingMiddleware)
	r.Use(CORSMiddleware)
	r.Use(RecoveryMiddleware)

	// Install logger for package-level helpers and middleware
	SetLogger(logger)

	// Create handlers
	systemHandler := NewSystemHandler(version, buildTime, database, aiEngine)
	authHandler := NewAuthHandler(repo.Engineer, repo.Profile, cfg.JWTSecret, cfg.TokenDuration)
	activitiesHandler := NewActivitiesHandler(repo.Activity, repo.Job)
	aiHandler := NewAIHandler(repo.Schema, repo.Template, aiEngine)

	// Open endpoints
	r.HandleFunc("/version", systemHandler.VersionHandler).Methods("GET")
	r.HandleFunc("/health", systemHandler.HealthHandler).Methods("GET")
	r.HandleFunc("/ready", systemHandler.ReadinessHandler).Methods("GET")
	r.HandleFunc("/live", systemHandler.LiveHandler).Methods("GET")
	r.HandleFunc("/v1/auth/signup", authHandler.Signup).Methods("POST")
	r.HandleFunc("/v1/auth/signin", authHandler.Signin).Methods("POST")

	// API v1 Protected routes
	apiV1 := r.PathPrefix("/v1").Subrouter()
	apiV1.Use(JWTAuthMiddlewareWithSecret(cfg.JWTSecret))

	// Auth endpoints
	authV1 := apiV1.PathPrefix("/auth").Subrouter()
	authV1.HandleFunc("/signout", authHandler.Signout).Methods("POST")

	// Activities endpoints
	activitiesV1 := apiV1.PathPrefix("/activities").Subrouter()
	activitiesV1.HandleFunc("", activitiesHandler.CreateActivity).Methods("POST")
	activitiesV1.HandleFunc("", activitiesHandler.ListActivities).Methods("GET")

	// AI management endpoints
	aiV1 := apiV1.PathPrefix("/ai").Subrouter()

	// AI schema endpoints
	schemaV1 := aiV1.PathPrefix("/schemas").Subrouter()
	schemaV1.HandleFunc("", aiHandler.ListSchemasHandler).Methods("GET")
	schemaV1.HandleFunc("", aiHandler.CreateOrUpdateSchemaHandler).Methods("POST")
	schemaV1.HandleFunc("/get", aiHandler.GetSchemaHandler).Methods("GET")
	schemaV1.HandleFunc("/delete", aiHandler.DeleteSchemaHandler).Methods("DELETE")
	schemaV1.HandleFunc("/reload", aiHandler.ReloadHandler).Methods("POST")

	// AI template endpoints
	templateV1 := aiV1.PathPrefix("/templates").Subrouter()
	templateV1.HandleFunc("", aiHandler.ListTemplatesHandler).Methods("GET")
	templateV1.HandleFunc("", aiHandler.CreateOrUpdateTemplateHandler).Methods("POST")
	templateV1.HandleFunc("/get", aiHandler.GetTemplateHandler).Methods("GET")
	templateV1.HandleFunc("/delete", aiHandler.DeleteTemplateHandler).Methods("DELETE")

	return r
}

func writeJSON(w http.ResponseWriter, v any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	enc := json.NewEncoder(w)
	if err := enc.Encode(v); err != nil {
		// use package logger if available
		logger.Error("Error writing JSON response", slog.Any("err", err), slog.Any("payload", v))
	}
}
