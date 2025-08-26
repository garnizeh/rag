# ai.process_response job

This document describes the `ai.process_response` background job used by the worker pool to process LLM inference results, merge them into an engineer's context, persist history, and create clarification questions for conflicts.

## Job type

- `ai.process_response`

## Payload shape (JSON)

The job payload is a JSON object with the following fields:

- `engineer_id` (integer): the engineer id the AI response applies to.
- `response` (object): the parsed `AIResponse` structure produced by the LLM or by `ai.Engine.ParseAIResponse`.

Example payload:

```json
{
  "engineer_id": 123,
  "response": {
    "version": "v1",
    "summary": "Worked on Project Phoenix",
    "entities": {
      "people": ["Alice"],
      "projects": ["Project Phoenix"],
      "technologies": ["Go", "SQLite"]
    },
    "confidence": 0.92,
    "context_update": true,
    "reasoning": "Project description and roles",
    "raw": "...original model output..."
  }
}
```

Notes:
- `response` must be a valid representation of the `ai.AIResponse` struct defined in `internal/ai/ai.go`.
- The worker handler will unmarshal the payload, convert `response` into an `ai.AIResponse` and call `ai.ProcessAIResponse`.

## Enqueueing a job (WorkerPool)

If you have a running `jobs.WorkerPool`, you can enqueue a job using its `Enqueue` helper. Example in Go:

```go
payload := map[string]any{
    "engineer_id": engineerID,
    "response":    aiResponseJSON, // json.RawMessage or marshalled bytes
}
if _, err := pool.Enqueue(ctx, "ai.process_response", payload, 10, 3); err != nil {
    // handle error
}
```

`pool.Enqueue` will marshal the payload to JSON and insert a `jobs` row. The worker pool will pick it up and dispatch to the `ai.process_response` handler.

## Enqueueing via repository (DB-level)

If you want to insert a job directly through the jobs repository (for example in APIs that don't have the pool object), call `jobs.NewRepository(db).Enqueue(ctx, job)` where `job` is a `jobs.Job` with `Type: "ai.process_response"` and `Payload` containing the payload JSON.

## Processing semantics

- The handler will call `ai.ProcessAIResponse(ctx, repo, engineerID, &resp)` internally. That function:
  - Merges AI response into the existing engineer context (`MergeAIResponse`).
  - Persists the merged context via `repo.Context.UpsertEngineerContext`.
  - Records a detailed history entry via `repo.Context.CreateContextHistory`.
  - Creates a clarification question via `repo.Question.CreateQuestion` if conflicts are detected.

- Retries: the worker handles retries with exponential backoff. If the handler returns an error, the job will be retried until `MaxAttempts` is reached and then moved to the dead letter queue.

## Security and validation

- Validate `engineer_id` and ensure the calling code has permissions to create processing jobs for that engineer.
- Prefer sending a validated `ai.AIResponse` object (parsed and schema-validated) in the `response` field.

## Troubleshooting

- If jobs are not processed, ensure the WorkerPool is running and that the `ai.process_response` handler is registered (see `cmd/server/main.go`).
- For debugging, inspect `engineer_contexts` and `engineer_context_history` tables to see persisted data.

***