package ollama

import "time"

// Config holds settings for the Ollama client and model management.
type Config struct {
	// BaseURL is the HTTP endpoint for the Ollama instance, e.g. http://localhost:11434
	BaseURL string `yaml:"base_url" json:"base_url"`
	// DefaultModelNames is a list of known model names to manage by default
	DefaultModelNames []string `yaml:"models" json:"models"`
	// Timeout is the per-request timeout
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
	// Retries is number of retry attempts for transient failures
	Retries int `yaml:"retries" json:"retries"`
	// Backoff is the base backoff between retries
	Backoff time.Duration `yaml:"backoff" json:"backoff"`
	// CircuitFailureThreshold opens circuit after this many consecutive failures
	CircuitFailureThreshold int `yaml:"circuit_failure_threshold" json:"circuit_failure_threshold"`
	// CircuitReset is the duration after which the circuit attempts to half-open
	CircuitReset time.Duration `yaml:"circuit_reset" json:"circuit_reset"`
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{
		BaseURL:                 "http://localhost:11434",
		DefaultModelNames:       []string{"deepseek-r1:32b", "llama3"},
		Timeout:                 30 * time.Second,
		Retries:                 3,
		Backoff:                 500 * time.Millisecond,
		CircuitFailureThreshold: 5,
		CircuitReset:            30 * time.Second,
	}
}
