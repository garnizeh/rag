package repository

import (
	"context"

	"github.com/garnizeh/rag/internal/models"
)

type Repository struct {
	Engineer EngineerRepo
	Profile  ProfileRepo
	Activity ActivityRepo
	Question QuestionRepo
	Job      JobRepo
	Context  ContextRepo
	Schema   SchemaRepo
	Template TemplateRepo
}

// Repository interfaces for domain entities. These are the public contracts
// consumers should depend on; concrete implementations live under internal/.

type EngineerRepo interface {
	CreateEngineer(ctx context.Context, e *models.Engineer) (int64, error)
	GetByID(ctx context.Context, id int64) (*models.Engineer, error)
	GetByEmail(ctx context.Context, email string) (*models.Engineer, error)
	UpdateEngineer(ctx context.Context, e *models.Engineer) error
	DeleteEngineer(ctx context.Context, id int64) error
}

type ProfileRepo interface {
	CreateProfile(ctx context.Context, p *models.Profile) (int64, error)
	GetByEngineerID(ctx context.Context, engineerID int64) (*models.Profile, error)
	UpdateProfile(ctx context.Context, p *models.Profile) error
	DeleteProfile(ctx context.Context, id int64) error
}

type ActivityRepo interface {
	CreateActivity(ctx context.Context, a *models.Activity) (int64, error)
	ListByEngineer(ctx context.Context, engineerID int64, limit, offset int) ([]models.Activity, error)
	CountActivitiesByEngineer(ctx context.Context, engineerID int64) (int64, error)
}

type QuestionRepo interface {
	CreateQuestion(ctx context.Context, q *models.Question) (int64, error)
	ListUnansweredByEngineer(ctx context.Context, engineerID int64) ([]models.Question, error)
}

type JobRepo interface {
	CreateJob(ctx context.Context, j *models.Job) (int64, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	Enqueue(ctx context.Context, j *models.BackgroundJob) (int64, error)
	FetchNext(ctx context.Context) (*models.BackgroundJob, error)
	UpdateJob(ctx context.Context, j *models.BackgroundJob) error
	MoveToDeadLetter(ctx context.Context, j *models.BackgroundJob) error
}

type SchemaRepo interface {
	CreateSchema(ctx context.Context, version, description, schemaJSON string) (int64, error)
	GetSchemaByVersion(ctx context.Context, version string) (*models.Schema, error)
	ListSchemas(ctx context.Context) ([]models.Schema, error)
	DeleteSchema(ctx context.Context, version string) error
}

type TemplateRepo interface {
	CreateTemplate(ctx context.Context, name, version, templateText string, schemaVersion *string, metadata *string) (int64, error)
	GetTemplate(ctx context.Context, name, version string) (*models.Template, error)
	ListTemplates(ctx context.Context) ([]models.Template, error)
	DeleteTemplate(ctx context.Context, name, version string) error
}

type ContextRepo interface {
	UpsertEngineerContext(ctx context.Context, engineerID int64, contextJSON string, appliedBy string) (int64, error)
	GetEngineerContext(ctx context.Context, engineerID int64) (string, int64, error)
	CreateContextHistory(ctx context.Context, engineerID int64, contextJSON string, changesJSON *string, conflictsJSON *string, appliedBy string, version int64) (int64, error)
	ListContextHistory(ctx context.Context, engineerID int64) ([]models.ContextHistory, error)
	GetContextHistoryByID(ctx context.Context, engineerID int64, historyID int64) (*models.ContextHistory, error)
}
