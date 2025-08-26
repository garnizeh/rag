package ai_test

import (
	"context"
	"errors"
	"testing"

	"github.com/garnizeh/rag/internal/ai"
	"github.com/garnizeh/rag/internal/models"
	"github.com/garnizeh/rag/pkg/repository"
)

// fakeSchemaRepo is a small in-memory implementation of repository.SchemaRepo for tests.
type fakeSchemaRepo struct {
	schemas map[string]models.Schema
}

func newFakeSchemaRepo() *fakeSchemaRepo {
	return &fakeSchemaRepo{schemas: make(map[string]models.Schema)}
}

func (f *fakeSchemaRepo) CreateSchema(ctx context.Context, version, description, schemaJSON string) (int64, error) {
	id := int64(len(f.schemas) + 1)
	f.schemas[version] = models.Schema{ID: id, Version: version, Description: description, SchemaJSON: schemaJSON}
	return id, nil
}

func (f *fakeSchemaRepo) GetSchemaByVersion(ctx context.Context, version string) (*models.Schema, error) {
	if s, ok := f.schemas[version]; ok {
		return &s, nil
	}
	return nil, nil
}

func (f *fakeSchemaRepo) ListSchemas(ctx context.Context) ([]models.Schema, error) {
	out := make([]models.Schema, 0, len(f.schemas))
	for _, s := range f.schemas {
		out = append(out, s)
	}
	return out, nil
}

func (f *fakeSchemaRepo) DeleteSchema(ctx context.Context, version string) error {
	if _, ok := f.schemas[version]; !ok {
		return errors.New("not found")
	}
	delete(f.schemas, version)
	return nil
}

// Ensure fakeSchemaRepo implements repository.SchemaRepo
var _ repository.SchemaRepo = (*fakeSchemaRepo)(nil)

func TestLoader_ReloadAndGetSchema_Success(t *testing.T) {
	fr := newFakeSchemaRepo()
	// minimal valid schema requiring 'version' field
	schema := `{"$schema":"http://json-schema.org/draft-07/schema#","type":"object","required":["version"],"properties":{"version":{"type":"string"}}}`
	if _, err := fr.CreateSchema(context.Background(), "v1", "v1 schema", schema); err != nil {
		t.Fatalf("seed schema failed: %v", err)
	}

	l, err := ai.NewLoader(context.Background(), fr)
	if err != nil {
		t.Fatalf("NewLoader error: %v", err)
	}
	s, ok := l.GetSchema("v1")
	if !ok || s == nil {
		t.Fatalf("expected schema in cache for v1")
	}

	// validate a matching document
	verrs, err := s.ValidateBytes(context.Background(), []byte(`{"version":"v1"}`))
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}
	if len(verrs) != 0 {
		t.Fatalf("expected no validation errors, got: %v", verrs)
	}
}
