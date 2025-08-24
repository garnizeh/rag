package main

import (
	"fmt"
	"io"
	"os"

	"github.com/garnizeh/rag/internal/config"
)

func main() {
	cfg, err := config.LoadConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}
	src := cfg.DatabasePath
	dst := src + ".bak"

	srcFile, err := os.Open(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Backup error: %v\n", err)
		os.Exit(1)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Backup error: %v\n", err)
		os.Exit(1)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Backup error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Database backup completed.")
}
