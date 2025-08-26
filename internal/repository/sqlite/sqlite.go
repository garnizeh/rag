package sqlite

import (
	"time"

	"log/slog"

	"github.com/garnizeh/rag/internal/db"
	"github.com/garnizeh/rag/pkg/repository"
)

// SQLiteRepo implements repository interfaces using the internal DB wrapper.
type SQLiteRepo struct {
	conn   *db.DB
	logger *slog.Logger
}

// Ensure SQLiteRepo implements the public interfaces.
var _ repository.EngineerRepo = (*SQLiteRepo)(nil)
var _ repository.ProfileRepo = (*SQLiteRepo)(nil)
var _ repository.ActivityRepo = (*SQLiteRepo)(nil)
var _ repository.QuestionRepo = (*SQLiteRepo)(nil)
var _ repository.JobRepo = (*SQLiteRepo)(nil)
var _ repository.ContextRepo = (*SQLiteRepo)(nil)
var _ repository.SchemaRepo = (*SQLiteRepo)(nil)
var _ repository.TemplateRepo = (*SQLiteRepo)(nil)

func New(conn *db.DB, logger *slog.Logger) *SQLiteRepo {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(nil, nil))
	}
	return &SQLiteRepo{conn: conn, logger: logger}
}

func now() int64 {
	return time.Now().UTC().UnixMilli()
}
