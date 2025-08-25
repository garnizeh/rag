package main

import (
	"context"
	"fmt"
	"os"

	dbfs "github.com/garnizeh/rag/db"
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
	database, err := db.New(ctx, cfg.DatabasePath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DB init error: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// run migrations and seed using internal/db.Migrate
	if err := db.Migrate(ctx, database, dbfs.Migrations, dbfs.SeedFiles); err != nil {
		fmt.Fprintf(os.Stderr, "Migration runner error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Database initialized successfully.")
}
