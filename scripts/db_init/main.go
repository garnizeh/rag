package main

import (
	"context"
	"fmt"
	"os"

	"github.com/garnizeh/rag/internal/config"
	"github.com/garnizeh/rag/internal/db"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}
	database, err := db.New(ctx, cfg.DatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DB init error: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// apply base migration
	migrationSQL, err := os.ReadFile("db/migrations/0001_init.sql")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migration file error: %v\n", err)
		os.Exit(1)
	}

	_, err = database.Exec(ctx, string(migrationSQL))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migration exec error: %v\n", err)
		os.Exit(1)
	}

	// apply AI schemas/templates migration
	migration2, err := os.ReadFile("db/migrations/0002_schemas_templates.sql")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migration2 file error: %v\n", err)
		os.Exit(1)
	}

	_, err = database.Exec(ctx, string(migration2))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migration2 exec error: %v\n", err)
		os.Exit(1)
	}

	// seed initial schema (v1) if not present
	initialSchema := `{
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
	}`

	// use INSERT OR REPLACE to upsert by version
	_, err = database.Exec(ctx, `INSERT OR REPLACE INTO ai_schemas (version, description, schema_json, created, updated) VALUES ('v1', 'default v1 schema', ?, strftime('%s','now'), strftime('%s','now'))`, initialSchema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Schema seed error: %v\n", err)
		os.Exit(1)
	}

	// seed default template for activity v1
	initialTemplate := `You are an assistant that analyzes short activity logs and returns a strict JSON object.
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
`

	schemaVer := "v1"
	metadata := `{"owner":"system","description":"default activity template"}`
	_, err = database.Exec(ctx, `INSERT OR REPLACE INTO ai_templates (name, version, template_text, schema_version, metadata, created, updated) VALUES ('activity', 'v1', ?, ?, ?, strftime('%s','now'), strftime('%s','now'))`, initialTemplate, schemaVer, metadata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Template seed error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Database initialized successfully.")
}
