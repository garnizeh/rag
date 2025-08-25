package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Addr           string        `yaml:"addr"`
	JWTSecret      string        `yaml:"jwt_secret"`
	DatabasePath   string        `yaml:"database_path"`
	APITimeout     time.Duration `yaml:"timeout"`
	TokenDuration  time.Duration `yaml:"token_duration"`
	MigrateOnStart bool          `yaml:"migrate_on_start"`
	EngineConfig   EngineConfig  `yaml:"engine"`
	Ollama         OllamaConfig  `yaml:"ollama"`
}

type EngineConfig struct {
	Model           string        `yaml:"model"`
	TemplateVersion string        `yaml:"template_version"`
	Timeout         time.Duration `yaml:"timeout"`
	MinConfidence   float64       `yaml:"min_confidence"`
}

type OllamaConfig struct {
	BaseURL                 string        `yaml:"base_url"`
	DefaultModelNames       []string      `yaml:"models"`
	Timeout                 time.Duration `yaml:"timeout"`
	Retries                 int           `yaml:"retries"`
	Backoff                 time.Duration `yaml:"backoff"`
	CircuitFailureThreshold int           `yaml:"circuit_failure_threshold"`
	CircuitReset            time.Duration `yaml:"circuit_reset"`
}

func LoadConfig(path string) (*Config, error) {
	apiTimeout := 15 * time.Second
	tokenDuration := 1 * time.Hour

	cfg := &Config{
		Addr:           getEnv("RAG_ADDR", ":8080"),
		JWTSecret:      getEnv("RAG_JWT_SECRET", "supersecretkey"),
		APITimeout:     apiTimeout,
		DatabasePath:   getEnv("RAG_DATABASE_PATH", "rag.db"),
		TokenDuration:  tokenDuration,
		MigrateOnStart: false,
	}
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		dec := yaml.NewDecoder(f)
		if err := dec.Decode(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// Validate ensures required configuration values are present and fills sensible defaults
// for optional fields. It returns an error when a required or insecure value is found.
func (c *Config) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("addr must be set")
	}
	if c.APITimeout <= 0 {
		return fmt.Errorf("api timeout must be > 0")
	}
	if c.DatabasePath == "" {
		return fmt.Errorf("database_path must be set")
	}
	if c.TokenDuration <= 0 {
		return fmt.Errorf("token_duration must be > 0")
	}
	if c.EngineConfig.Model == "" {
		// engine model is required for AI features
		return fmt.Errorf("engine.model must be set")
	}

	// Reject insecure default JWT secret in non-development environments
	if c.JWTSecret == "" || c.JWTSecret == "supersecretkey" {
		if os.Getenv("RAG_ENV") != "development" {
			return fmt.Errorf("jwt_secret is not set or is insecure; set RAG_JWT_SECRET")
		}
	}

	// Provide sensible defaults for Ollama config if not supplied
	if c.Ollama.BaseURL == "" {
		c.Ollama.BaseURL = "http://localhost:11434"
	}
	if c.Ollama.Timeout == 0 {
		c.Ollama.Timeout = 30 * time.Second
	}
	if c.Ollama.Retries == 0 {
		c.Ollama.Retries = 3
	}
	if c.Ollama.Backoff == 0 {
		c.Ollama.Backoff = 500 * time.Millisecond
	}
	if c.Ollama.CircuitFailureThreshold == 0 {
		c.Ollama.CircuitFailureThreshold = 5
	}
	if c.Ollama.CircuitReset == 0 {
		c.Ollama.CircuitReset = 30 * time.Second
	}
	if len(c.Ollama.DefaultModelNames) == 0 {
		c.Ollama.DefaultModelNames = []string{"deepseek-r1:32b", "llama3"}
	}

	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}
