package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/garnizeh/rag/pkg/models"
	"github.com/garnizeh/rag/pkg/ollama"

	"github.com/qri-io/jsonschema"
)

// PromptTemplate holds a versioned prompt template.
type PromptTemplate struct {
	Version  string
	Template string
	Example  string
}

// DefaultActivityTemplate is a simple initial prompt template (versioned).
var DefaultActivityTemplate = PromptTemplate{
	Version: "v1",
	Template: `You are an assistant that analyzes short activity logs and returns a strict JSON object.
Return only a single JSON object. The JSON must conform to the following fields:
- version: string (template version)
- summary: short string summarizing the activity
- entities: { people: [string], projects: [string], technologies: [string] }
- confidence: number between 0.0 and 1.0
- context_update: boolean (true if this activity should change stored context)
- reasoning: explanation of how you arrived at the answer

Activity: {{.Activity.Activity}}
Context: {{.Context}}

Example:
{
  "version": "v1",
  "summary": "Updated README with deployment notes",
  "entities": {"people":["Alice"], "projects":["deploy-svc"], "technologies":["Docker"]},
  "confidence": 0.92,
  "context_update": true,
  "reasoning": "Activity mentions deployment and Docker, likely relevant to deploy-svc."
}
`,
	Example: "see template",
}

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

// EngineConfig configures the AI engine behavior.
type EngineConfig struct {
	Model         string
	Template      PromptTemplate
	Timeout       time.Duration
	MinConfidence float64
}

// Engine wraps an Ollama client and provides analysis helpers.
type Engine struct {
	client *ollama.Client
	cfg    EngineConfig
}

// NewEngine creates a new AI engine.
func NewEngine(client *ollama.Client, cfg EngineConfig) *Engine {
	// apply sensible defaults
	if cfg.Template.Version == "" {
		cfg.Template = DefaultActivityTemplate
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 20 * time.Second
	}
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = 0.5
	}

	return &Engine{client: client, cfg: cfg}
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

	// assess confidence
	assessed := AssessConfidence(resp)
	if resp.Confidence == nil {
		resp.Confidence = &assessed
	}

	// validate
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

	// Validate against JSON Schema (basic inline schema)
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["version","summary","entities","context_update","reasoning"],
		"properties": {
			"version": {"type":"string"},
			"summary": {"type":"string"},
			"entities": {
				"type":"object",
				"properties": {
					"people": {"type":"array","items":{"type":"string"}},
					"projects": {"type":"array","items":{"type":"string"}},
					"technologies": {"type":"array","items":{"type":"string"}}
				}
			},
			"confidence": {"type":"number"},
			"context_update": {"type":"boolean"},
			"reasoning": {"type":"string"}
		}
	}`)

	rs := &jsonschema.Schema{}
	if err := json.Unmarshal(schemaJSON, rs); err != nil {
		// schema load error: log and continue (don't fail parsing on internal schema problems)
		log.Printf("ai schema load error: %v", err)
		return &r, nil
	}

	// validate the raw JSON map
	var raw interface{}
	if err := json.Unmarshal([]byte(j), &raw); err != nil {
		return &r, nil // should not happen since we already unmarshaled
	}

	verrs, err := rs.ValidateBytes(context.Background(), []byte(j))
	if err != nil {
		// validation execution error: log and allow
		log.Printf("ai schema validate error: %v", err)
		return &r, nil
	}
	if len(verrs) > 0 {
		// collect messages
		var sb strings.Builder
		for _, v := range verrs {
			sb.WriteString(v.Message)
			sb.WriteString("; ")
		}
		return &r, fmt.Errorf("response does not match schema: %s", sb.String())
	}

	// success
	_ = raw
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
