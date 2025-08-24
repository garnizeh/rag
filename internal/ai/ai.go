package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

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
	client *ollama.Client
	cfg    config.EngineConfig
	loader *Loader
}

// NewEngine creates a new AI engine. Loader is required for schema validation.
func NewEngine(ctx context.Context, client *ollama.Client, cfg config.EngineConfig, sr repository.SchemaRepo, tr repository.TemplateRepo) (*Engine, error) {
	// apply sensible defaults
	if cfg.Template.Version == "" {
		cfg.Template.Version = "v1"
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
	tpl, terr := tr.GetTemplate(ctx, "activity", cfg.Template.Version)
	if terr != nil {
		return nil, fmt.Errorf("load template: %w", terr)
	}
	if tpl == nil || tpl.TemplateTxt == "" {
		return nil, fmt.Errorf("template activity:%s not found", cfg.Template.Version)
	}
	cfg.Template.Template = tpl.TemplateTxt
	cfg.Template.Version = tpl.Version
	cfg.Template.SchemaVersion = tpl.SchemaVer

	return &Engine{client: client, cfg: cfg, loader: loader}, nil
}

// AnalyzeActivity renders a prompt for an activity, sends it to Ollama, and parses the structured response.
func (e *Engine) AnalyzeActivity(ctx context.Context, activity models.Activity, contextText string) (*AIResponse, error) {
	// prepare prompt
	data := map[string]any{"Activity": activity, "Context": contextText}
	prompt, err := ollama.RenderTemplate(e.cfg.Template.Template, data)
	if err != nil {
		return nil, fmt.Errorf("render template: %w", err)
	}

	// call LLM with timeout
	ctxReq, cancel := context.WithTimeout(ctx, e.cfg.Timeout)
	defer cancel()

	out, err := e.client.Generate(ctxReq, e.cfg.Model, prompt)
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}

	// parse response

	resp, perr := ParseAIResponse(out)
	if perr != nil {
		// log raw output for debugging and return a structured error
		log.Printf("ai parse error: %v; raw=%s", perr, out)
		return nil, fmt.Errorf("parse response: %w", perr)
	}
	resp.Raw = out

	// fill missing version
	if resp.Version == "" {
		resp.Version = e.cfg.Template.Version
	}

	// validate against loader-provided schema
	j := extractJSON(out)
	if j == "" {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	// prefer template's schema_version if provided
	schemaVer := resp.Version
	if e.cfg.Template.SchemaVersion != nil && *e.cfg.Template.SchemaVersion != "" {
		schemaVer = *e.cfg.Template.SchemaVersion
	}

	schema, ok := e.loader.GetSchema(schemaVer)
	if !ok || schema == nil {
		return nil, fmt.Errorf("no schema found for version %s", schemaVer)
	}

	verrs, err := schema.ValidateBytes(ctxReq, []byte(j))
	if err != nil {
		log.Printf("ai schema validate error: %v", err)
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
		log.Printf("low confidence (%.2f) for activity id=%d", *resp.Confidence, activity.ID)
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
