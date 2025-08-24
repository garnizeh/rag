package repository

import (
	"context"

	"github.com/garnizeh/rag/pkg/models"
)

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
}
