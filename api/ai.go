package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/garnizeh/rag/internal/ai"
	"github.com/garnizeh/rag/pkg/repository"
	"github.com/qri-io/jsonschema"
)

type AIHandler struct {
	schemaRepo   repository.SchemaRepo
	templateRepo repository.TemplateRepo
	engine       *ai.Engine
}

func NewAIHandler(
	schemaRepo repository.SchemaRepo,
	templateRepo repository.TemplateRepo,
	engine *ai.Engine,
) *AIHandler {
	return &AIHandler{
		schemaRepo:   schemaRepo,
		templateRepo: templateRepo,
		engine:       engine,
	}
}

func (h *AIHandler) ReloadHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.engine.ReloadSchemas(r.Context()); err != nil {
		http.Error(w, fmt.Sprintf("reload schemas: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AIHandler) ListSchemasHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := h.schemaRepo.ListSchemas(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("list schemas: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, rows, http.StatusOK)
}

type schemaPayload struct {
	Version     string          `json:"version"`
	Description string          `json:"description,omitempty"`
	SchemaJSON  json.RawMessage `json:"schema_json"`
}

// CreateOrUpdateSchemaHandler validates and stores a schema
func (h *AIHandler) CreateOrUpdateSchemaHandler(w http.ResponseWriter, r *http.Request) {
	var p schemaPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if p.Version == "" {
		http.Error(w, "version required", http.StatusBadRequest)
		return
	}

	// basic compile check using qri-io/jsonschema
	rs := &jsonschema.Schema{}
	if err := json.Unmarshal(p.SchemaJSON, rs); err != nil {
		http.Error(w, fmt.Sprintf("invalid schema json: %v", err), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	if _, err := rs.ValidateBytes(ctx, p.SchemaJSON); err != nil {
		// ValidateBytes returns execution error; treat as bad schema
		http.Error(w, fmt.Sprintf("schema compile error: %v", err), http.StatusBadRequest)
		return
	}

	if _, err := h.schemaRepo.CreateSchema(ctx, p.Version, p.Description, string(p.SchemaJSON)); err != nil {
		http.Error(w, fmt.Sprintf("store schema: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSchemaHandler returns a single schema by version (expects ?version=...)
func (h *AIHandler) GetSchemaHandler(w http.ResponseWriter, r *http.Request) {
	version := r.URL.Query().Get("version")
	if version == "" {
		http.Error(w, "version required", http.StatusBadRequest)
		return
	}

	s, err := h.schemaRepo.GetSchemaByVersion(r.Context(), version)
	if err != nil {
		http.Error(w, fmt.Sprintf("get schema: %v", err), http.StatusInternalServerError)
		return
	}
	if s == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	writeJSON(w, s, http.StatusOK)
}

// DeleteSchemaHandler deletes schema by version (expects ?version=...)
func (h *AIHandler) DeleteSchemaHandler(w http.ResponseWriter, r *http.Request) {
	version := r.URL.Query().Get("version")
	if version == "" {
		http.Error(w, "version required", http.StatusBadRequest)
		return
	}

	if err := h.schemaRepo.DeleteSchema(r.Context(), version); err != nil {
		http.Error(w, fmt.Sprintf("delete schema: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListTemplatesHandler returns all templates
func (h *AIHandler) ListTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := h.templateRepo.ListTemplates(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("list templates: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, rows, http.StatusOK)
}

type templatePayload struct {
	Name        string  `json:"name"`
	Version     string  `json:"version"`
	TemplateTxt string  `json:"template_text"`
	SchemaVer   *string `json:"schema_version,omitempty"`
}

// CreateOrUpdateTemplateHandler stores a template, enforcing size limit
func (h *AIHandler) CreateOrUpdateTemplateHandler(w http.ResponseWriter, r *http.Request) {
	// limit read to 64KB
	const maxSize = 64 * 1024
	body, err := io.ReadAll(io.LimitReader(r.Body, maxSize+1))
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	if len(body) > maxSize {
		http.Error(w, "template too large", http.StatusBadRequest)
		return
	}

	var p templatePayload
	if err := json.Unmarshal(body, &p); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if p.Name == "" || p.Version == "" || p.TemplateTxt == "" {
		http.Error(w, "name, version and template_text required", http.StatusBadRequest)
		return
	}

	if _, err := h.templateRepo.CreateTemplate(r.Context(), p.Name, p.Version, p.TemplateTxt, p.SchemaVer, nil); err != nil {
		http.Error(w, fmt.Sprintf("store template: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetTemplateHandler returns one template by query params name and version
func (h *AIHandler) GetTemplateHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	version := r.URL.Query().Get("version")
	if name == "" || version == "" {
		http.Error(w, "name and version required", http.StatusBadRequest)
		return
	}

	t, err := h.templateRepo.GetTemplate(r.Context(), name, version)
	if err != nil {
		http.Error(w, fmt.Sprintf("get template: %v", err), http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	writeJSON(w, t, http.StatusOK)
}

// DeleteTemplateHandler deletes a template
func (h *AIHandler) DeleteTemplateHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	version := r.URL.Query().Get("version")
	if name == "" || version == "" {
		http.Error(w, "name and version required", http.StatusBadRequest)
		return
	}

	if err := h.templateRepo.DeleteTemplate(r.Context(), name, version); err != nil {
		http.Error(w, fmt.Sprintf("delete template: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
