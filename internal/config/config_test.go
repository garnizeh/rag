package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/garnizeh/rag/internal/config"
)

func TestValidate_InsecureJWT_FailsWhenNotDevelopment(t *testing.T) {
	os.Setenv("RAG_ENV", "production")
	defer os.Unsetenv("RAG_ENV")

	cfg := &config.Config{
		Addr:          ":8080",
		JWTSecret:     "supersecretkey",
		APITimeout:    5 * time.Second,
		DatabasePath:  "rag.db",
		TokenDuration: 1 * time.Hour,
		EngineConfig:  config.EngineConfig{Model: "m"},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected Validate to fail for insecure JWT in non-development env")
	}
}

func TestValidate_InsecureJWT_AllowsDevelopment(t *testing.T) {
	os.Setenv("RAG_ENV", "development")
	defer os.Unsetenv("RAG_ENV")

	cfg := &config.Config{
		Addr:          ":8080",
		JWTSecret:     "supersecretkey",
		APITimeout:    5 * time.Second,
		DatabasePath:  "rag.db",
		TokenDuration: 1 * time.Hour,
		EngineConfig:  config.EngineConfig{Model: "m"},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected Validate to succeed in development env, got: %v", err)
	}
}

func TestValidate_MissingEngineModel(t *testing.T) {
	os.Setenv("RAG_ENV", "development")
	defer os.Unsetenv("RAG_ENV")

	cfg := &config.Config{
		Addr:          ":8080",
		JWTSecret:     "strongsecret",
		APITimeout:    5 * time.Second,
		DatabasePath:  "rag.db",
		TokenDuration: 1 * time.Hour,
		EngineConfig:  config.EngineConfig{Model: ""},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected Validate to fail when engine.model is empty")
	}
}

func TestValidate_OllamaDefaultsPopulated(t *testing.T) {
	os.Setenv("RAG_ENV", "development")
	defer os.Unsetenv("RAG_ENV")

	cfg := &config.Config{
		Addr:          ":8080",
		JWTSecret:     "strongsecret",
		APITimeout:    5 * time.Second,
		DatabasePath:  "rag.db",
		TokenDuration: 1 * time.Hour,
		EngineConfig:  config.EngineConfig{Model: "m"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate failed unexpectedly: %v", err)
	}

	if cfg.Ollama.BaseURL == "" {
		t.Fatalf("expected Ollama.BaseURL to be populated, got empty")
	}
	if cfg.Ollama.Timeout <= 0 {
		t.Fatalf("expected Ollama.Timeout to be > 0")
	}
	if cfg.Ollama.Retries == 0 {
		t.Fatalf("expected Ollama.Retries default to be non-zero")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Ensure environment does not interfere
	_ = os.Unsetenv("RAG_ADDR")
	_ = os.Unsetenv("RAG_JWT_SECRET")
	_ = os.Unsetenv("RAG_DATABASE_PATH")

	cfg, err := config.LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig returned error for empty path: %v", err)
	}

	if cfg.Addr != ":8080" {
		t.Fatalf("unexpected Addr: got %q want %q", cfg.Addr, ":8080")
	}
	if cfg.JWTSecret != "supersecretkey" {
		t.Fatalf("unexpected JWTSecret: got %q want %q", cfg.JWTSecret, "supersecretkey")
	}
	if cfg.DatabasePath != "rag.db" {
		t.Fatalf("unexpected DatabasePath: got %q want %q", cfg.DatabasePath, "rag.db")
	}
	if cfg.APITimeout != 15*time.Second {
		t.Fatalf("unexpected APITimeout: got %v want %v", cfg.APITimeout, 15*time.Second)
	}
	if cfg.TokenDuration != 1*time.Hour {
		t.Fatalf("unexpected TokenDuration: got %v want %v", cfg.TokenDuration, 1*time.Hour)
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

	cfg, err := config.LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("LoadConfig returned error for file: %v", err)
	}

	if cfg.Addr != ":9090" {
		t.Fatalf("unexpected Addr: got %q want %q", cfg.Addr, ":9090")
	}
	if cfg.JWTSecret != "filekey" {
		t.Fatalf("unexpected JWTSecret: got %q want %q", cfg.JWTSecret, "filekey")
	}
	if cfg.DatabasePath != "test.db" {
		t.Fatalf("unexpected DatabasePath: got %q want %q", cfg.DatabasePath, "test.db")
	}
	if cfg.APITimeout != 30*time.Second {
		t.Fatalf("unexpected APITimeout: got %v want %v", cfg.APITimeout, 30*time.Second)
	}
	if cfg.TokenDuration != 2*time.Hour {
		t.Fatalf("unexpected TokenDuration: got %v want %v", cfg.TokenDuration, 2*time.Hour)
	}
}

func TestLoadConfig_BadPath(t *testing.T) {
	if _, err := config.LoadConfig("/path/that/does/not/exist.yaml"); err == nil {
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

	if _, err := config.LoadConfig(f.Name()); err == nil {
		t.Fatalf("expected YAML decode error, got nil")
	}
}

func TestValidate_InsecureJWT(t *testing.T) {
	// ensure not in development
	_ = os.Unsetenv("RAG_ENV")

	cfg := &config.Config{
		Addr:          ":8080",
		JWTSecret:     "supersecretkey",
		APITimeout:    15 * time.Second,
		DatabasePath:  "rag.db",
		TokenDuration: 1 * time.Hour,
		EngineConfig: config.EngineConfig{
			Model: "test-model",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for insecure jwt secret, got nil")
	}
}
