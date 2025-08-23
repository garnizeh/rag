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
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	var (
		addr = flag.String("addr", ":8080", "HTTP server address")
		help = flag.Bool("help", false, "Show help message")
		ver  = flag.Bool("version", false, "Show version information")
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

	log.Printf("Starting RAG server version %s (built at %s)", version, buildTime)

	// Create HTTP server
	server := &http.Server{
		Addr:         *addr,
		Handler:      setupRoutes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", *addr)
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

func setupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok","service":"rag"}`)
	})

	// Version endpoint
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"version":"%s","buildTime":"%s"}`, version, buildTime)
	})

	// TODO: Add RAG endpoints here

	return mux
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
