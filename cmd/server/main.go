package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/garnizeh/rag/api"
	"github.com/garnizeh/rag/internal/config"
	"github.com/garnizeh/rag/internal/db"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	var configPath = flag.String("config", "", "Path to config YAML file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting RAG server version %s (built at %s)", version, buildTime)

	ctx := context.Background()

	// Open database connection
	db, err := db.New(ctx, cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}

	handler := api.SetupRoutes(cfg, version, buildTime, db)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.APITimeout,
		WriteTimeout: cfg.APITimeout,
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
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close database connection
	if err := db.Close(); err != nil {
		log.Printf("Error closing DB: %v", err)
	}

	log.Println("Server exited")
}
