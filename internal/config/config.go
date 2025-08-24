package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Addr          string        `yaml:"addr"`
	JWTSecret     string        `yaml:"jwt_secret"`
	APITimeout    time.Duration `yaml:"timeout"`
	DatabasePath  string        `yaml:"database_path"`
	TokenDuration time.Duration `yaml:"token_duration"`
	EngineConfig  EngineConfig  `yaml:"engine"`
}

type EngineConfig struct {
	Model         string         `yaml:"model"`
	Template      PromptTemplate `yaml:"template"`
	Timeout       time.Duration  `yaml:"timeout"`
	MinConfidence float64        `yaml:"min_confidence"`
}

type PromptTemplate struct {
	Version       string  `yaml:"version"`
	Template      string  `yaml:"template"`
	Example       string  `yaml:"example"`
	SchemaVersion *string `yaml:"schema_version,omitempty"`
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
		Addr:          getEnv("RAG_ADDR", ":8080"),
		JWTSecret:     getEnv("RAG_JWT_SECRET", "supersecretkey"),
		APITimeout:    apiTimeout,
		DatabasePath:  getEnv("RAG_DATABASE_PATH", "rag.db"),
		TokenDuration: tokenDuration,
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

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}
