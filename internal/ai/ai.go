package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/garnizeh/rag/internal/config"
	"github.com/garnizeh/rag/pkg/models"
	"github.com/garnizeh/rag/pkg/ollama"
	"github.com/garnizeh/rag/pkg/repository"
)

// AIResponse represents the structured response we expect from the LLM.
type AIResponse struct {
	Version  string `json:"version"`
	Summary  string `json:"summary"`
	Entities struct {
		People       []string `json:"people"`
		Projects     []string `json:"projects"`
		Technologies []string `json:"technologies"`
	} `json:"entities"`
	Confidence    *float64 `json:"confidence,omitempty"`
	ContextUpdate bool     `json:"context_update"`
	Reasoning     string   `json:"reasoning"`

	// Raw captures the original model output for auditing/logging.
	Raw string `json:"-"`
}

// Engine wraps an Ollama client and provides analysis helpers.
type Engine struct {
	cfg                   config.EngineConfig
	loader                *Loader
	templateText          string
	templateSchemaVersion *string
	mu                    sync.RWMutex
	client                *ollama.Client
}

// package logger for ai; can be set by callers via SetLogger
var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// SetLogger sets the logger used by internal/ai package. Passing nil is a no-op.
func SetLogger(l *slog.Logger) {
	if l != nil {
		logger = l
	}
}

// NewEngine creates a new AI engine. Loader is required for schema validation.
func NewEngine(ctx context.Context, client *ollama.Client, cfg config.EngineConfig, sr repository.SchemaRepo, tr repository.TemplateRepo) (*Engine, error) {
	// apply sensible defaults
	if cfg.TemplateVersion == "" {
		cfg.TemplateVersion = "v1"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 20 * time.Second
	}
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = 0.5
	}

	if sr == nil {
		return nil, fmt.Errorf("schema repo is required")
	}

	if tr == nil {
		return nil, fmt.Errorf("template repo is required")
	}

	loader, err := NewLoader(ctx, sr)
	if err != nil {
		return nil, fmt.Errorf("create loader: %w", err)
	}

	// load template for activity from DB; fail if not present
	tpl, terr := tr.GetTemplate(ctx, "activity", cfg.TemplateVersion)
	if terr != nil {
		return nil, fmt.Errorf("load template: %w", terr)
	}
	if tpl == nil || tpl.TemplateTxt == "" {
		return nil, fmt.Errorf("template activity:%s not found", cfg.TemplateVersion)
	}
	// store selected template version into cfg so other methods can access it
	cfg.TemplateVersion = tpl.Version
	// populate engine template fields
	templateSchema := tpl.SchemaVer
	// attach schema version to a new field by reusing loader via template metadata
	// we'll rely on loader + schema repo for validation at runtime using tpl.SchemaVer when needed

	// persist template text by setting an internal field on Engine if needed; keep cfg minimal
	// (Engine will read tpl.TemplateTxt when producing prompts)

	eng := &Engine{client: client, cfg: cfg, loader: loader, templateText: tpl.TemplateTxt, templateSchemaVersion: templateSchema}

	return eng, nil
}

// AnalyzeActivity renders a prompt for an activity, sends it to Ollama, and parses the structured response.
func (e *Engine) AnalyzeActivity(ctx context.Context, activity models.Activity, contextText string) (*AIResponse, error) {
	// If client is nil we are running in degraded mode (no LLM). Return a clear error
	// so callers can handle or fallback to raw processing. Use a read lock to allow
	// a background probe to replace the client concurrently.
	e.mu.RLock()
	client := e.client
	e.mu.RUnlock()
	if client == nil {
		return nil, fmt.Errorf("llm unavailable: engine running in degraded mode")
	}
	// prepare prompt
	data := map[string]any{"Activity": activity, "Context": contextText}
	prompt, err := ollama.RenderTemplate(e.templateText, data)
	if err != nil {
		return nil, fmt.Errorf("render template: %w", err)
	}

	// call LLM with timeout
	ctxReq, cancel := context.WithTimeout(ctx, e.cfg.Timeout)
	defer cancel()

	res, err := client.Generate(ctxReq, e.cfg.Model, prompt)
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}

	// parse response
	resp, perr := ParseAIResponse(res.Text)
	if perr != nil {
		// log raw output for debugging and return a structured error
		logger.Error("ai parse error", slog.Any("err", perr), slog.String("raw", res.Text))
		return nil, fmt.Errorf("parse response: %w", perr)
	}
	// store raw textual output for auditing
	resp.Raw = res.Text

	// fill missing version
	if resp.Version == "" {
		resp.Version = e.cfg.TemplateVersion
	}

	// validate against loader-provided schema
	j := extractJSON(res.Text)
	if j == "" {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	// prefer template's schema_version if provided
	schemaVer := resp.Version
	if e.templateSchemaVersion != nil && *e.templateSchemaVersion != "" {
		schemaVer = *e.templateSchemaVersion
	}

	schema, ok := e.loader.GetSchema(schemaVer)
	if !ok || schema == nil {
		return nil, fmt.Errorf("no schema found for version %s", schemaVer)
	}

	verrs, err := schema.ValidateBytes(ctxReq, []byte(j))
	if err != nil {
		logger.Error("ai schema validate error", slog.Any("err", err))
		return nil, fmt.Errorf("schema validate error: %w", err)
	}
	if len(verrs) > 0 {
		var sb strings.Builder
		for _, v := range verrs {
			sb.WriteString(v.Message)
			sb.WriteString("; ")
		}
		return nil, fmt.Errorf("response does not match schema: %s", sb.String())
	}

	// assess confidence
	assessed := AssessConfidence(resp)
	if resp.Confidence == nil {
		resp.Confidence = &assessed
	}

	// validate confidence threshold

	if *resp.Confidence < e.cfg.MinConfidence {
		logger.Warn("low confidence for activity", slog.Float64("confidence", *resp.Confidence), slog.Int64("activity_id", activity.ID))
	}

	// final safety checks
	if resp.Entities.People == nil {
		resp.Entities.People = []string{}
	}
	if resp.Entities.Projects == nil {
		resp.Entities.Projects = []string{}
	}
	if resp.Entities.Technologies == nil {
		resp.Entities.Technologies = []string{}
	}

	return resp, nil
}

func (e *Engine) ReloadSchemas(ctx context.Context) error {
	return e.loader.Reload(ctx)
}

// SetClient updates the underlying Ollama client in a thread-safe manner.
// Passing nil puts the engine into degraded mode.
func (e *Engine) SetClient(c *ollama.Client) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.client = c
}

// Available reports whether an LLM client is currently configured.
func (e *Engine) Available() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.client != nil
}

// ParseAIResponse tries to extract a JSON object from arbitrary model output and unmarshal it.
func ParseAIResponse(s string) (*AIResponse, error) {
	if strings.TrimSpace(s) == "" {
		return nil, errors.New("empty response")
	}

	j := extractJSON(s)
	if j == "" {
		return nil, errors.New("no JSON object found in response")
	}

	var r AIResponse
	if err := json.Unmarshal([]byte(j), &r); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	return &r, nil
}

// extractJSON returns the substring from the first '{' to the last '}' in the input.
// This is a pragmatic approach to handle model outputs that wrap JSON in text or markdown.
func extractJSON(s string) string {
	first := strings.Index(s, "{")
	last := strings.LastIndex(s, "}")
	if first == -1 || last == -1 || last < first {
		return ""
	}
	return s[first : last+1]
}

// AssessConfidence returns a simple confidence score when one is not provided.
// Currently this is heuristic: presence of summary and at least one entity increases confidence.
func AssessConfidence(r *AIResponse) float64 {
	score := 0.0
	if strings.TrimSpace(r.Summary) != "" {
		score += 0.4
	}
	if len(r.Entities.People)+len(r.Entities.Projects)+len(r.Entities.Technologies) > 0 {
		score += 0.4
	}
	if strings.TrimSpace(r.Reasoning) != "" {
		score += 0.2
	}
	if score > 1.0 {
		score = 1.0
	}
	return score
}
