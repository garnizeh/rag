package main

import (
	"context"
	"fmt"
	"os"

	"github.com/garnizeh/rag/internal/config"
	"github.com/garnizeh/rag/internal/db"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}
	database, err := db.New(ctx, cfg.DatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DB init error: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	migrationSQL, err := os.ReadFile("db/migrations/0001_init.sql")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migration file error: %v\n", err)
		os.Exit(1)
	}

	_, err = database.Exec(ctx, string(migrationSQL))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migration exec error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Database initialized successfully.")
}
