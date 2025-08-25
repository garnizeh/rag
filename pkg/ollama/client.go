package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"github.com/garnizeh/rag/internal/config"
	"github.com/ollama/ollama/api"
)

var ErrCircuitOpen = errors.New("ollama circuit open")

// Client wraps the Ollama API client and adds retries, timeout, and circuit breaker.
type Client struct {
	api    *api.Client
	cfg    config.OllamaConfig
	client *http.Client

	// simple circuit breaker state
	failures  int32
	openUntil int64 // unix nano
	closed    int32 // atomic flag for Close()
}

// GenerateResult is a typed representation of a model response.
type GenerateResult struct {
	Text string          `json:"text"`
	Raw  json.RawMessage `json:"raw"`
	Meta map[string]any  `json:"meta,omitempty"`
}

// NewClient creates a new Ollama client wrapper.
func NewClient(cfg config.OllamaConfig, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}

	u, err := url.ParseRequestURI(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	c := &Client{
		api:    api.NewClient(u, httpClient),
		cfg:    cfg,
		client: httpClient,
	}
	// use package logger
	logger.Info("ollama: NewClient created", slog.String("base_url", cfg.BaseURL), slog.Duration("timeout", cfg.Timeout))
	return c, nil
}

// Instrumentation: log client creation for easier debugging
func init() {
	// no-op init to keep package initialization explicit; logging happens in NewClient
	_ = 0
}

func NewDefaultClient(cfg config.OllamaConfig) (*Client, error) {
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

// Close releases any resources held by the client. Currently this will close
// idle connections on the underlying HTTP transport when supported. Close is
// idempotent and safe to call multiple times.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	// ensure we only run close once
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return nil
	}
	if c.client != nil && c.client.Transport != nil {
		if tr, ok := c.client.Transport.(interface{ CloseIdleConnections() }); ok {
			tr.CloseIdleConnections()
			logger.Info("ollama: client Close() called - CloseIdleConnections invoked")
		}
	}
	return nil
}

// package-level logger for pkg/ollama; can be replaced by callers
var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// SetLogger sets the logger used by pkg/ollama. Passing nil is a no-op.
func SetLogger(l *slog.Logger) {
	if l != nil {
		logger = l
	}
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
func (c *Client) Generate(ctx context.Context, model string, prompt string) (GenerateResult, error) {
	var lastErr error
	var empty GenerateResult
	if c.isCircuitOpen() {
		return empty, ErrCircuitOpen
	}

	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		ctxReq, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
		req := &api.GenerateRequest{Model: model, Prompt: prompt}
		var lastRaw any
		var outText string
		var lastRawB []byte
		start := time.Now()
		err := c.api.Generate(ctxReq, req, func(r api.GenerateResponse) error {
			// store raw response and a textual representation
			lastRaw = r
			// prefer a stable JSON representation of the response as text
			if b, merr := json.Marshal(r); merr == nil {
				lastRawB = b
				outText = string(b)
			} else {
				// fallback to a formatted string if marshalling fails
				outText = fmt.Sprintf("%+v", r)
			}
			return nil
		})

		cancel()
		latency := time.Since(start)
		if err == nil {
			// marshal raw into JSON for Raw field (reuse lastRawB if we created it)
			var rawB []byte
			if lastRawB != nil {
				rawB = lastRawB
			} else {
				rawB, _ = json.Marshal(lastRaw)
			}
			atomic.StoreInt32(&c.failures, 0)
			meta := map[string]any{"model": model, "latency_ms": latency.Milliseconds()}
			return GenerateResult{Text: outText, Raw: rawB, Meta: meta}, nil
		}

		lastErr = err
		c.recordFailure()

		// backoff
		time.Sleep(c.cfg.Backoff * time.Duration(attempt+1))
		if c.isCircuitOpen() {
			return empty, ErrCircuitOpen
		}
	}

	return empty, fmt.Errorf("generate failed after retries: %w", lastErr)
}
