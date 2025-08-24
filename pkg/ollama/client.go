package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/ollama/ollama/api"
)

var ErrCircuitOpen = errors.New("ollama circuit open")

// Client wraps the Ollama API client and adds retries, timeout, and circuit breaker.
type Client struct {
	api    *api.Client
	cfg    Config
	client *http.Client

	// simple circuit breaker state
	failures  int32
	openUntil int64 // unix nano
}

// NewClient creates a new Ollama client wrapper.
func NewClient(cfg Config, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}

	u, err := url.ParseRequestURI(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	return &Client{
		api:    api.NewClient(u, httpClient),
		cfg:    cfg,
		client: httpClient,
	}, nil
}

func NewDefaultClient(cfg Config) (*Client, error) {
	defaultClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 15 * time.Second,
				DualStack: true,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	return NewClient(cfg, defaultClient)
}

func (c *Client) isCircuitOpen() bool {
	if atomic.LoadInt32(&c.failures) < int32(c.cfg.CircuitFailureThreshold) {
		return false
	}

	if time.Now().UnixNano() < atomic.LoadInt64(&c.openUntil) {
		return true
	}

	// attempt half-open: reset failures and allow a request
	atomic.StoreInt32(&c.failures, 0)
	return false
}

func (c *Client) recordFailure() {
	v := atomic.AddInt32(&c.failures, 1)
	if v >= int32(c.cfg.CircuitFailureThreshold) {
		atomic.StoreInt64(&c.openUntil, time.Now().Add(c.cfg.CircuitReset).UnixNano())
	}
}

// Health pings the Ollama instance by requesting info about models.
func (c *Client) Health(ctx context.Context) error {
	if c.isCircuitOpen() {
		return ErrCircuitOpen
	}

	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	// list models via HTTP API
	models, err := c.ListModels(ctx)
	if err != nil {
		c.recordFailure()
		return fmt.Errorf("health check failed: %w", err)
	}
	if len(models) == 0 {
		c.recordFailure()
		return fmt.Errorf("health check failed: no models returned")
	}

	// success: reset failures
	atomic.StoreInt32(&c.failures, 0)
	return nil
}

// ListModels returns model metadata from Ollama.
// ModelInfo is a lightweight model descriptor returned by ListModels.
type ModelInfo struct {
	Name string          `json:"name"`
	Raw  json.RawMessage `json:"-"`
}

// ListModels calls the Ollama /models endpoint and returns basic model info.
func (c *Client) ListModels(ctx context.Context) ([]ModelInfo, error) {
	if c.isCircuitOpen() {
		return nil, ErrCircuitOpen
	}

	// build URL: base + /models
	base, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	u := base.ResolveReference(&url.URL{Path: "/models"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		c.recordFailure()
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.recordFailure()
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.recordFailure()
		return nil, fmt.Errorf("models endpoint returned status %d", resp.StatusCode)
	}

	var raw []map[string]any
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&raw); err != nil {
		c.recordFailure()
		return nil, err
	}

	out := make([]ModelInfo, 0, len(raw))
	for _, m := range raw {
		name := ""
		if v, ok := m["name"].(string); ok {
			name = v
		}
		b, _ := json.Marshal(m)
		out = append(out, ModelInfo{Name: name, Raw: b})
	}

	atomic.StoreInt32(&c.failures, 0)
	return out, nil
}

// Generate sends a prompt to the model and collects the response. It supports retries and timeouts.
// Generate sends a prompt to Ollama and returns a string representation of the response.
func (c *Client) Generate(ctx context.Context, model string, prompt string) (string, error) {
	var lastErr error
	if c.isCircuitOpen() {
		return "", ErrCircuitOpen
	}

	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		ctxReq, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
		// cancel on next loop
		req := &api.GenerateRequest{Model: model, Prompt: prompt}
		var out string
		err := c.api.Generate(ctxReq, req, func(r api.GenerateResponse) error {
			out = fmt.Sprintf("%+v", r)
			return nil
		})

		cancel()
		if err == nil {
			atomic.StoreInt32(&c.failures, 0)
			return out, nil
		}

		lastErr = err
		c.recordFailure()

		// backoff
		time.Sleep(c.cfg.Backoff * time.Duration(attempt+1))
		if c.isCircuitOpen() {
			return "", ErrCircuitOpen
		}
	}

	return "", fmt.Errorf("generate failed after retries: %w", lastErr)
}
