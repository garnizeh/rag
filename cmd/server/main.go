package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/garnizeh/rag/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	var (
		configPath = flag.String("config", "", "Path to config YAML file")
		help       = flag.Bool("help", false, "Show help message")
		ver        = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	if *ver {
		showVersion()
		return
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting RAG server version %s (built at %s)", version, buildTime)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      setupRoutesWithConfig(cfg),
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRoutesWithConfig(cfg *config.Config) http.Handler {
	r := mux.NewRouter()

	// Middleware chain
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)
	r.Use(recoveryMiddleware)

	// JWT Auth middleware for protected routes (stub)
	api := r.PathPrefix("/v1").Subrouter()
	api.Use(jwtAuthMiddlewareWithSecret(cfg.JWTSecret))

	// Health check endpoint
	r.HandleFunc("/v1/system/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok","service":"rag"}`)
	}).Methods("GET")

	// Version endpoint
	r.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"version":"%s","buildTime":"%s"}`, version, buildTime)
	}).Methods("GET")

	// TODO: Add other API endpoints here

	return r
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Recovery middleware
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// JWT Auth middleware (stub)
func jwtAuthMiddlewareWithSecret(secret string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}
			var tokenString string
			fmt.Sscanf(authHeader, "Bearer %s", &tokenString)
			if tokenString == "" {
				http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
				return
			}
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}
			// Token is valid, continue
			next.ServeHTTP(w, r)
		})
	}
}

func showHelp() {
	fmt.Println("RAG Server - Retrieval-Augmented Generation System")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  rag-server [options]")
	fmt.Println("")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  rag-server                    # Start server on default port 8080")
	fmt.Println("  rag-server -addr :9000        # Start server on port 9000")
	fmt.Println("  rag-server -version           # Show version information")
}

func showVersion() {
	fmt.Printf("RAG Server\n")
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Built: %s\n", buildTime)
}
