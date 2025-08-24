package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/garnizeh/rag/pkg/repository"
	"github.com/qri-io/jsonschema"
)

// Loader loads and caches compiled JSON schemas from the repository.
type Loader struct {
	repo  repository.SchemaRepo
	mu    sync.RWMutex
	cache map[string]*jsonschema.Schema
}

func NewLoader(ctx context.Context, r repository.SchemaRepo) (*Loader, error) {
	l := &Loader{
		repo:  r,
		cache: make(map[string]*jsonschema.Schema),
	}
	// initial load
	if err := l.Reload(ctx); err != nil {
		return nil, err
	}

	return l, nil
}

// GetSchema returns a compiled schema for a version.
func (l *Loader) GetSchema(version string) (*jsonschema.Schema, bool) {
	l.mu.RLock()
	s, ok := l.cache[version]
	l.mu.RUnlock()

	return s, ok
}

// Reload loads all schemas from the DB and compiles them.
func (l *Loader) Reload(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	rows, err := l.repo.ListSchemas(ctx)
	if err != nil {
		return fmt.Errorf("load schemas: %w", err)
	}

	newCache := make(map[string]*jsonschema.Schema)
	for _, r := range rows {
		rs := &jsonschema.Schema{}
		if err := json.Unmarshal([]byte(r.SchemaJSON), rs); err != nil {
			return fmt.Errorf("compile schema %s: %w", r.Version, err)
		}

		newCache[r.Version] = rs
	}

	l.cache = newCache
	return nil
}
