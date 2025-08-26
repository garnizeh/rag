package mock

import (
	"context"

	"github.com/garnizeh/rag/internal/models"
)

// Test helpers and mocks
type Mocks struct {
	EngRepo  *mockEngineerRepo
	ProfRepo *mockProfileRepo
}

func NewMocks() *Mocks {
	return &Mocks{
		EngRepo:  &mockEngineerRepo{},
		ProfRepo: &mockProfileRepo{},
	}
}

type mockEngineerRepo struct {
	Stored    *models.Engineer
	CreateErr error
}

func (m *mockEngineerRepo) CreateEngineer(ctx context.Context, e *models.Engineer) (int64, error) {
	if m.CreateErr != nil {
		return 0, m.CreateErr
	}
	m.Stored = &models.Engineer{ID: 1, Name: e.Name, Email: e.Email, PasswordHash: e.PasswordHash}
	return 1, nil
}

func (m *mockEngineerRepo) GetByID(ctx context.Context, id int64) (*models.Engineer, error) {
	if m.Stored != nil && m.Stored.ID == id {
		return m.Stored, nil
	}
	return nil, nil
}

func (m *mockEngineerRepo) GetByEmail(ctx context.Context, email string) (*models.Engineer, error) {
	if m.Stored != nil && m.Stored.Email == email {
		return m.Stored, nil
	}
	return nil, nil
}

func (m *mockEngineerRepo) UpdateEngineer(ctx context.Context, e *models.Engineer) error {
	m.Stored = e
	return nil
}

func (m *mockEngineerRepo) DeleteEngineer(ctx context.Context, id int64) error {
	if m.Stored != nil && m.Stored.ID == id {
		m.Stored = nil
	}
	return nil
}

type mockProfileRepo struct{}

func (m *mockProfileRepo) CreateProfile(ctx context.Context, p *models.Profile) (int64, error) {
	return 1, nil
}

func (m *mockProfileRepo) GetByEngineerID(ctx context.Context, engineerID int64) (*models.Profile, error) {
	return nil, nil
}

func (m *mockProfileRepo) UpdateProfile(ctx context.Context, p *models.Profile) error { return nil }

func (m *mockProfileRepo) DeleteProfile(ctx context.Context, id int64) error { return nil }
