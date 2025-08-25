package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"
	"sync"

	"github.com/garnizeh/rag/api"
	dbfs "github.com/garnizeh/rag/db"
	"github.com/garnizeh/rag/internal/ai"
	"github.com/garnizeh/rag/internal/config"
	"github.com/garnizeh/rag/internal/db"
	"github.com/garnizeh/rag/internal/repository/sqlite"
	"github.com/garnizeh/rag/pkg/ollama"
	"github.com/garnizeh/rag/pkg/repository"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	var configPath = flag.String("config", "", "Path to config YAML file")
	flag.Parse()

	// Always use JSON handler for structured logs (production-friendly single format)
	slogHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(slogHandler)

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Error("Failed to load config", slog.Any("err", err))
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		logger.Error("Invalid configuration", slog.Any("err", err))
		os.Exit(1)
	}

	logger.Info("Starting RAG server", slog.String("version", version), slog.String("buildTime", buildTime))

	rootCtx := context.Background()

	// Open database connection with a bounded timeout to fail-fast on DB problems
	dbCtx, dbCancel := context.WithTimeout(rootCtx, cfg.APITimeout)
	defer dbCancel()
	database, err := db.New(dbCtx, cfg.DatabasePath, logger)
	if err != nil {
		logger.Error("Failed to open DB", slog.Any("err", err))
		os.Exit(1)
	}

	// Optionally run migrations on start (opt-in)
	if cfg.MigrateOnStart {
		logger.Info("migrate_on_start enabled - applying migrations and seeds")
		// run migrations with the same bounded DB context to ensure quick failure on issues
		if err := db.Migrate(dbCtx, database, dbfs.Migrations, dbfs.SeedFiles); err != nil {
			logger.Error("Migration runner error", slog.Any("err", err))
			os.Exit(1)
		}
		logger.Info("migrations and seeds applied")
	}

	// Repository
	sqliteRepo := sqlite.New(database, logger)
	repo := repository.Repository{
		Engineer: sqliteRepo,
		Profile:  sqliteRepo,
		Activity: sqliteRepo,
		Question: sqliteRepo,
		Job:      sqliteRepo,
		Schema:   sqliteRepo,
		Template: sqliteRepo,
	}

	// Ollama client
	var client *ollama.Client
	var clientMu sync.Mutex
	{
		c, err := ollama.NewDefaultClient(cfg.Ollama)
		if err != nil {
			// Do not fail startup for Ollama client creation errors: operate in degraded mode
			logger.Warn("failed to create Ollama client, running in degraded mode", slog.Any("err", err))
			client = nil
		} else {
			// Quick health check for Ollama client — if it fails, continue in degraded mode.
			// use a bounded context based on configured Ollama timeout so we don't hang startup
			ollamaCtx, ollamaCancel := context.WithTimeout(rootCtx, cfg.Ollama.Timeout)
			if herr := c.Health(ollamaCtx); herr != nil {
				logger.Warn("Ollama health check failed, running in degraded mode", slog.Any("err", herr))
				// close client immediately to free resources
				_ = c.Close()
				ollamaCancel()
				client = nil
			} else {
				ollamaCancel()
				clientMu.Lock()
				client = c
				clientMu.Unlock()
			}
		}
	}

	// AI engine
	aiEngine, err := ai.NewEngine(rootCtx, client, cfg.EngineConfig, sqliteRepo, sqliteRepo)
	if err != nil {
		logger.Error("Failed to initialize AI engine", slog.Any("err", err))
		os.Exit(1)
	}
	// propagate logger into AI subsystem for consistent structured logs
	ai.SetLogger(logger)

	// Background probe: periodically try to (re)create and health-check Ollama client
	go func() {
		ticker := time.NewTicker(cfg.Ollama.Backoff)
		defer ticker.Stop()
		for {
			select {
			case <-rootCtx.Done():
				return
			case <-ticker.C:
				// attempt to create new client and health-check
				c, cerr := ollama.NewDefaultClient(cfg.Ollama)
				if cerr != nil {
					// can't create client; set engine to degraded
					aiEngine.SetClient(nil)
					logger.Warn("Ollama probe: client create failed", slog.Any("err", cerr))
					// close any old client we may have stored
					clientMu.Lock()
					old := client
					client = nil
					clientMu.Unlock()
					if old != nil {
						_ = old.Close()
					}
					continue
				}
				probeCtx, probeCancel := context.WithTimeout(rootCtx, cfg.Ollama.Timeout)
				if herr := c.Health(probeCtx); herr != nil {
					aiEngine.SetClient(nil)
					logger.Warn("Ollama probe: health failed", slog.Any("err", herr))
					probeCancel()
					// close newly created client
					_ = c.Close()
					// ensure stored client is nil
					clientMu.Lock()
					old := client
					client = nil
					clientMu.Unlock()
					if old != nil {
						_ = old.Close()
					}
					continue
				}
				probeCancel()
				// success: update engine client and swap stored client
				clientMu.Lock()
				old := client
				client = c
				clientMu.Unlock()
				aiEngine.SetClient(c)
				logger.Info("Ollama probe: client healthy, engine updated")
				if old != nil {
					_ = old.Close()
				}
			}
		}
	}()

	handler := api.SetupRoutes(cfg, version, buildTime, repo, aiEngine, database, logger)

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
		logger.Info("Server starting", slog.String("addr", cfg.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server")

	// Give outstanding requests 30 seconds to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(rootCtx, 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", slog.Any("err", err))
		os.Exit(1)
	}

	// Close database connection
	if err := database.Close(); err != nil {
		logger.Warn("Error closing DB", slog.Any("err", err))
	}

	// Close Ollama client if present
	clientMu.Lock()
	if client != nil {
		_ = client.Close()
		client = nil
	}
	clientMu.Unlock()

	logger.Info("Server exited")
}
