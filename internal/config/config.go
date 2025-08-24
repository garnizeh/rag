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
