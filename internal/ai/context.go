package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ContextModel is a lightweight generic representation of engineer context stored as JSON.
// Keep small to avoid coupling to DB models; callers convert as needed.
type ContextModel map[string]any

// ChangeRecord represents a single change applied to the context for audit and rollback.
type ChangeRecord struct {
	Key       string `json:"key"`
	OldValue  any    `json:"old_value,omitempty"`
	NewValue  any    `json:"new_value,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// MergeResult holds the result of merging AIResponse into an existing context.
type MergeResult struct {
	Merged    ContextModel   `json:"merged"`
	Changes   []ChangeRecord `json:"changes"`
	Conflicts []string       `json:"conflicts"`
}

// ErrInvalidEntityName is returned when an entity name fails validation.
var ErrInvalidEntityName = errors.New("invalid entity name")

// ValidateName performs simple validation on entity names.
// For now: non-empty, trimmed, and length < 256.
func ValidateName(s string) error {
	if strings.TrimSpace(s) == "" {
		return ErrInvalidEntityName
	}
	if len(s) > 255 {
		return ErrInvalidEntityName
	}
	return nil
}

// MergeAIResponse merges fields from AIResponse into the provided context JSON bytes.
// It returns the merged context, a list of changes, and list of detected conflicts.
// This function is pure and does not persist anything.
func MergeAIResponse(ctx context.Context, existingJSON []byte, resp *AIResponse) (*MergeResult, error) {
	// parse existing context
	var existing ContextModel
	if len(existingJSON) == 0 {
		existing = make(ContextModel)
	} else {
		if err := json.Unmarshal(existingJSON, &existing); err != nil {
			return nil, fmt.Errorf("parse existing context: %w", err)
		}
	}

	// make a copy to modify
	merged := make(ContextModel)
	for k, v := range existing {
		merged[k] = v
	}

	var changes []ChangeRecord
	var conflicts []string
	now := time.Now().UTC().Unix()

	// helper to set array-string fields (projects, people, technologies)
	setEntities := func(key string, items []string) {
		if len(items) == 0 {
			return
		}
		// validate
		valid := make([]string, 0, len(items))
		for _, it := range items {
			if err := ValidateName(it); err != nil {
				// skip invalid names but record conflict
				conflicts = append(conflicts, fmt.Sprintf("%s:invalid:%s", key, it))
				continue
			}
			valid = append(valid, strings.TrimSpace(it))
		}

		if len(valid) == 0 {
			return
		}

		// read existing
		if cur, ok := merged[key]; ok {
			// normalize to []string if possible
			switch cv := cur.(type) {
			case []any:
				// convert any->string
				seen := map[string]struct{}{}
				for _, a := range cv {
					if s, ok := a.(string); ok {
						seen[s] = struct{}{}
					}
				}
				// add new
				added := false
				for _, s := range valid {
					if _, found := seen[s]; !found {
						cv = append(cv, s)
						added = true
					}
				}
				if added {
					changes = append(changes, ChangeRecord{Key: key, OldValue: cur, NewValue: cv, Timestamp: now})
					merged[key] = cv
				}
			case []string:
				seen := map[string]struct{}{}
				for _, s := range cv {
					seen[s] = struct{}{}
				}
				added := false
				for _, s := range valid {
					if _, found := seen[s]; !found {
						cv = append(cv, s)
						added = true
					}
				}
				if added {
					changes = append(changes, ChangeRecord{Key: key, OldValue: cur, NewValue: cv, Timestamp: now})
					merged[key] = cv
				}
			default:
				// conflict: existing non-list value
				conflicts = append(conflicts, fmt.Sprintf("%s:existing_nonlist", key))
			}
		} else {
			// set new
			anyList := make([]any, len(valid))
			for i, s := range valid {
				anyList[i] = s
			}
			changes = append(changes, ChangeRecord{Key: key, OldValue: nil, NewValue: anyList, Timestamp: now})
			merged[key] = anyList
		}
	}

	// merge entities
	setEntities("people", resp.Entities.People)
	setEntities("projects", resp.Entities.Projects)
	setEntities("technologies", resp.Entities.Technologies)

	// merge summary into a 'summary' field if present
	if strings.TrimSpace(resp.Summary) != "" {
		if cur, ok := merged["summary"]; ok {
			// if summary changed, append as new value and record change
			if curStr, ok := cur.(string); ok {
				if strings.TrimSpace(curStr) != strings.TrimSpace(resp.Summary) {
					changes = append(changes, ChangeRecord{Key: "summary", OldValue: curStr, NewValue: resp.Summary, Timestamp: now})
					merged["summary"] = resp.Summary
				}
			} else {
				// conflict if existing is not string
				conflicts = append(conflicts, "summary:existing_nonstring")
			}
		} else {
			changes = append(changes, ChangeRecord{Key: "summary", OldValue: nil, NewValue: resp.Summary, Timestamp: now})
			merged["summary"] = resp.Summary
		}
	}

	// flag context_update intent from AI: put into metadata
	mergedMeta := map[string]any{}
	if mcur, ok := merged["_meta"]; ok {
		if mm, ok := mcur.(map[string]any); ok {
			for k, v := range mm {
				mergedMeta[k] = v
			}
		}
	}
	mergedMeta["last_ai_update"] = now
	mergedMeta["context_update_intent"] = resp.ContextUpdate
	merged["_meta"] = mergedMeta

	return &MergeResult{Merged: merged, Changes: changes, Conflicts: conflicts}, nil
}

// DiffContexts returns a JSON diff (list of ChangeRecord) between two context JSON blobs.
func DiffContexts(beforeJSON, afterJSON []byte) ([]ChangeRecord, error) {
	var before, after ContextModel
	if len(beforeJSON) == 0 {
		before = make(ContextModel)
	} else {
		if err := json.Unmarshal(beforeJSON, &before); err != nil {
			return nil, fmt.Errorf("parse before: %w", err)
		}
	}
	if len(afterJSON) == 0 {
		after = make(ContextModel)
	} else {
		if err := json.Unmarshal(afterJSON, &after); err != nil {
			return nil, fmt.Errorf("parse after: %w", err)
		}
	}

	var out []ChangeRecord
	now := time.Now().UTC().Unix()
	for k, av := range after {
		if bv, ok := before[k]; ok {
			// if equal (via marshal) skip
			bbs, _ := json.Marshal(bv)
			abs, _ := json.Marshal(av)
			if string(bbs) != string(abs) {
				out = append(out, ChangeRecord{Key: k, OldValue: bv, NewValue: av, Timestamp: now})
			}
		} else {
			out = append(out, ChangeRecord{Key: k, OldValue: nil, NewValue: av, Timestamp: now})
		}
	}
	// detect deletions
	for k, bv := range before {
		if _, ok := after[k]; !ok {
			out = append(out, ChangeRecord{Key: k, OldValue: bv, NewValue: nil, Timestamp: now})
		}
	}

	return out, nil
}
