package api

import (
	"github.com/garnizeh/rag/internal/config"
	"github.com/garnizeh/rag/internal/db"
	"github.com/garnizeh/rag/internal/repository/sqlite"
	"github.com/gorilla/mux"
)

func SetupRoutes(cfg *config.Config, version, buildTime string, db *db.DB) *mux.Router {
	r := mux.NewRouter()

	// Middleware chain
	r.Use(LoggingMiddleware)
	r.Use(CORSMiddleware)
	r.Use(RecoveryMiddleware)

	// Repository
	repo := sqlite.New(db)

	// Create handlers
	systemHandler := &SystemHandler{}
	authHandler := NewAuthHandler(repo, repo, cfg.JWTSecret, cfg.TokenDuration)

	// Open endpoints
	r.HandleFunc("/version", systemHandler.VersionHandler(version, buildTime)).Methods("GET")
	r.HandleFunc("/health", systemHandler.HealthHandler).Methods("GET")
	r.HandleFunc("/v1/auth/signup", authHandler.Signup).Methods("POST")
	r.HandleFunc("/v1/auth/signin", authHandler.Signin).Methods("POST")

	// API v1
	apiV1 := r.PathPrefix("/v1").Subrouter()

	// Protected routes
	apiV1.Use(JWTAuthMiddlewareWithSecret(cfg.JWTSecret))

	// Auth endpoints
	authV1 := apiV1.PathPrefix("/auth").Subrouter()
	authV1.HandleFunc("/signout", authHandler.Signout).Methods("POST")

	return r
}
