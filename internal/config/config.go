package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Addr      string        `yaml:"addr"`
	JWTSecret string        `yaml:"jwt_secret"`
	Timeout   time.Duration `yaml:"timeout"`
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{
		Addr:      getEnv("RAG_ADDR", ":8080"),
		JWTSecret: getEnv("RAG_JWT_SECRET", "supersecretkey"),
		Timeout:   15 * time.Second,
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
