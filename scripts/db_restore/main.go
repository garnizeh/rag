package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	src := "rag.db.bak"
	dst := "rag.db"

	srcFile, err := os.Open(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Restore error: %v\n", err)
		os.Exit(1)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Restore error: %v\n", err)
		os.Exit(1)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Restore error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Database restore completed.")
}
