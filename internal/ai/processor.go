package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"

	"github.com/garnizeh/rag/internal/models"
	"github.com/garnizeh/rag/pkg/repository"
)

var processorLogger = slog.New(slog.NewJSONHandler(nil, nil))

func SetProcessorLogger(l *slog.Logger) {
	if l != nil {
		processorLogger = l
	}
}

// ProcessAIResponse merges AI response into engineer context, persists changes,
// creates clarification question(s) for conflicts, and returns the applied version.
func ProcessAIResponse(ctx context.Context, repo *repository.Repository, engineerID int64, resp *AIResponse) (int64, error) {
	if repo == nil || repo.Context == nil {
		return 0, fmt.Errorf("repository.Context is required")
	}

	// get existing context
	existingJSON, _, err := repo.Context.GetEngineerContext(ctx, engineerID)
	if err != nil {
		return 0, fmt.Errorf("get existing context: %w", err)
	}

	// merge
	mr, merr := MergeAIResponse(ctx, []byte(existingJSON), resp)
	if merr != nil {
		return 0, fmt.Errorf("merge ai response: %w", merr)
	}

	mergedBytes, _ := json.Marshal(mr.Merged)

	// persist using repo.Context.UpsertEngineerContext which also records history
	version, err := repo.Context.UpsertEngineerContext(ctx, engineerID, string(mergedBytes), "ai")
	if err != nil {
		return 0, fmt.Errorf("persist merged context: %w", err)
	}

	// create a history item with changes and conflicts JSON
	if repo.Context != nil {
		// attach change/conflict details
		var changesJSON *string
		var conflictsJSON *string
		if len(mr.Changes) > 0 {
			if b, err := json.Marshal(mr.Changes); err == nil {
				s := string(b)
				changesJSON = &s
			}
		}
		if len(mr.Conflicts) > 0 {
			if b, err := json.Marshal(mr.Conflicts); err == nil {
				s := string(b)
				conflictsJSON = &s
			}
		}
		// CreateContextHistory to ensure details are stored (Upsert already inserts a history entry, but we call explicitly for richer payload)
		if _, cerr := repo.Context.CreateContextHistory(ctx, engineerID, string(mergedBytes), changesJSON, conflictsJSON, "ai", version); cerr != nil {
			processorLogger.Warn("create context history failed", "err", cerr)
		}
	}

	// handle conflicts: create clarification question(s)
	if len(mr.Conflicts) > 0 && repo.Question != nil {
		// build a concise question
		qText := fmt.Sprintf("I detected %d potential conflicts in your context: %v. Please confirm or clarify.", len(mr.Conflicts), mr.Conflicts)
		// use internal models.Question
		q := &models.Question{EngineerID: engineerID, Question: qText}
		if _, qerr := repo.Question.CreateQuestion(ctx, q); qerr != nil {
			processorLogger.Warn("create question failed", "err", qerr)
		}
	}

	processorLogger.Info("processed ai response", "engineer_id", engineerID, "version", version, "changes", len(mr.Changes), "conflicts", len(mr.Conflicts))

	return version, nil
}
