package config_test

import (
	"os"
	"testing"
	"time"

	cfg "github.com/garnizeh/rag/internal/config"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Ensure environment does not interfere
	_ = os.Unsetenv("RAG_ADDR")
	_ = os.Unsetenv("RAG_JWT_SECRET")
	_ = os.Unsetenv("RAG_DATABASE_PATH")

	c, err := cfg.LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig returned error for empty path: %v", err)
	}

	if c.Addr != ":8080" {
		t.Fatalf("unexpected Addr: got %q want %q", c.Addr, ":8080")
	}
	if c.JWTSecret != "supersecretkey" {
		t.Fatalf("unexpected JWTSecret: got %q want %q", c.JWTSecret, "supersecretkey")
	}
	if c.DatabasePath != "rag.db" {
		t.Fatalf("unexpected DatabasePath: got %q want %q", c.DatabasePath, "rag.db")
	}
	if c.APITimeout != 15*time.Second {
		t.Fatalf("unexpected APITimeout: got %v want %v", c.APITimeout, 15*time.Second)
	}
	if c.TokenDuration != 1*time.Hour {
		t.Fatalf("unexpected TokenDuration: got %v want %v", c.TokenDuration, 1*time.Hour)
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	// Create a temp YAML file with overrides
	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	content := []byte("addr: \":9090\"\njwt_secret: \"filekey\"\ntimeout: \"30s\"\ndatabase_path: \"test.db\"\ntoken_duration: \"2h\"\n")
	if err := os.WriteFile(f.Name(), content, 0o600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	c, err := cfg.LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("LoadConfig returned error for file: %v", err)
	}

	if c.Addr != ":9090" {
		t.Fatalf("unexpected Addr: got %q want %q", c.Addr, ":9090")
	}
	if c.JWTSecret != "filekey" {
		t.Fatalf("unexpected JWTSecret: got %q want %q", c.JWTSecret, "filekey")
	}
	if c.DatabasePath != "test.db" {
		t.Fatalf("unexpected DatabasePath: got %q want %q", c.DatabasePath, "test.db")
	}
	if c.APITimeout != 30*time.Second {
		t.Fatalf("unexpected APITimeout: got %v want %v", c.APITimeout, 30*time.Second)
	}
	if c.TokenDuration != 2*time.Hour {
		t.Fatalf("unexpected TokenDuration: got %v want %v", c.TokenDuration, 2*time.Hour)
	}
}

func TestLoadConfig_BadPath(t *testing.T) {
	if _, err := cfg.LoadConfig("/path/that/does/not/exist.yaml"); err == nil {
		t.Fatalf("expected error for nonexistent path, got nil")
	}
}

func TestLoadConfig_BadYAML(t *testing.T) {
	f, err := os.CreateTemp("", "bad-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	if err := os.WriteFile(f.Name(), []byte("::: not yaml :::"), 0o600); err != nil {
		t.Fatalf("failed to write bad yaml: %v", err)
	}

	if _, err := cfg.LoadConfig(f.Name()); err == nil {
		t.Fatalf("expected YAML decode error, got nil")
	}
}
