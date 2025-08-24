package sqlite_test

import (
	"context"
	"testing"
	"time"

	dbpkg "github.com/garnizeh/rag/internal/db"
	sqlite "github.com/garnizeh/rag/internal/repository/sqlite"
	"github.com/garnizeh/rag/pkg/models"
)

func setupRepo(t *testing.T) (*sqlite.SQLiteRepo, func()) {
	t.Helper()
	ctx := context.Background()
	d, err := dbpkg.New(ctx, "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	// create schema required by the repo
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS engineers (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, email TEXT, updated INTEGER, password_hash TEXT);`,
		`CREATE TABLE IF NOT EXISTS engineer_profiles (id INTEGER PRIMARY KEY AUTOINCREMENT, engineer_id INTEGER, bio TEXT, updated INTEGER);`,
		`CREATE TABLE IF NOT EXISTS raw_activities (id INTEGER PRIMARY KEY AUTOINCREMENT, engineer_id INTEGER, activity TEXT, created INTEGER);`,
		`CREATE TABLE IF NOT EXISTS ai_questions (id INTEGER PRIMARY KEY AUTOINCREMENT, engineer_id INTEGER, question TEXT, answered INTEGER, created INTEGER);`,
		`CREATE TABLE IF NOT EXISTS processing_jobs (id INTEGER PRIMARY KEY AUTOINCREMENT, status TEXT, created INTEGER);`,
	}

	for _, s := range stmts {
		if _, err := d.Exec(ctx, s); err != nil {
			d.Close()
			t.Fatalf("failed to exec schema: %v", err)
		}
	}

	repo := sqlite.New(d)
	return repo, func() { d.Close() }
}

func TestEngineerCRUD(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()
	ctx := context.Background()

	// nil engineer should error
	if _, err := repo.CreateEngineer(ctx, nil); err == nil {
		t.Fatalf("expected error when creating nil engineer")
	}

	// Non-existing ID should return nil, nil
	got, err := repo.GetByID(ctx, 9999)
	if err != nil {
		t.Fatalf("expected no error when getting non-existing ID")
	}
	if got != nil {
		t.Fatalf("expected nil when getting non-existing ID got: %#v", got)
	}

	// Non-existing email should return nil, nil
	got, err = repo.GetByEmail(ctx, "a@a.com")
	if err != nil {
		t.Fatalf("expected no error when getting non-existing email")
	}
	if got != nil {
		t.Fatalf("expected nil when getting non-existing email got: %#v", got)
	}

	e := &models.Engineer{Name: "Alice", Email: "alice@example.com", PasswordHash: "hash"}
	id, err := repo.CreateEngineer(ctx, e)
	if err != nil {
		t.Fatalf("CreateEngineer error: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected non-zero id")
	}

	got, err = repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if got == nil || got.Email != e.Email {
		t.Fatalf("GetByID wrong result: %#v", got)
	}

	byEmail, err := repo.GetByEmail(ctx, e.Email)
	if err != nil {
		t.Fatalf("GetByEmail error: %v", err)
	}
	if byEmail == nil || byEmail.ID != id {
		t.Fatalf("GetByEmail wrong result: %#v", byEmail)
	}

	// update
	got.Name = "Alice2"
	if err := repo.UpdateEngineer(ctx, got); err != nil {
		t.Fatalf("UpdateEngineer error: %v", err)
	}

	if err := repo.UpdateEngineer(ctx, nil); err == nil {
		t.Fatalf("expected error when updating nil engineer")
	}

	// delete
	if err := repo.DeleteEngineer(ctx, id); err != nil {
		t.Fatalf("DeleteEngineer error: %v", err)
	}

	after, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID after delete error: %v", err)
	}
	if after != nil {
		t.Fatalf("expected nil after delete got: %#v", after)
	}
}

func TestProfileCRUD(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()
	ctx := context.Background()

	// create engineer
	e := &models.Engineer{Name: "Bob", Email: "bob@example.com", PasswordHash: "h"}
	eid, err := repo.CreateEngineer(ctx, e)
	if err != nil {
		t.Fatalf("CreateEngineer error: %v", err)
	}

	// nil profile should error
	if _, err := repo.CreateProfile(ctx, nil); err == nil {
		t.Fatalf("expected error when creating nil profile")
	}

	p := &models.Profile{EngineerID: eid, Bio: "hello"}
	pid, err := repo.CreateProfile(ctx, p)
	if err != nil {
		t.Fatalf("CreateProfile error: %v", err)
	}
	if pid == 0 {
		t.Fatalf("expected profile id > 0")
	}

	got, err := repo.GetByEngineerID(ctx, eid)
	if err != nil {
		t.Fatalf("GetByEngineerID error: %v", err)
	}
	if got == nil || got.Bio != p.Bio {
		t.Fatalf("GetByEngineerID wrong: %#v", got)
	}

	// update
	if err := repo.UpdateProfile(ctx, nil); err == nil {
		t.Fatalf("expected error when updating nil profile")
	}

	got.Bio = "updated"
	if err := repo.UpdateProfile(ctx, got); err != nil {
		t.Fatalf("UpdateProfile error: %v", err)
	}

	if err := repo.DeleteProfile(ctx, pid); err != nil {
		t.Fatalf("DeleteProfile error: %v", err)
	}

	after, err := repo.GetByEngineerID(ctx, eid)
	if err != nil {
		t.Fatalf("GetByEngineerID after delete error: %v", err)
	}
	if after != nil {
		t.Fatalf("expected nil profile after delete got: %#v", after)
	}
}

func TestActivityAndQuestionList(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()
	ctx := context.Background()

	e := &models.Engineer{Name: "Carol", Email: "carol@example.com", PasswordHash: "p"}
	eid, err := repo.CreateEngineer(ctx, e)
	if err != nil {
		t.Fatalf("CreateEngineer error: %v", err)
	}

	// create activities
	if _, err := repo.CreateActivity(ctx, nil); err == nil {
		t.Fatalf("expected error when creating nil activity")
	}

	for range 3 {
		_, err := repo.CreateActivity(ctx, &models.Activity{EngineerID: eid, Activity: "act"})
		if err != nil {
			t.Fatalf("CreateActivity error: %v", err)
		}
		// small sleep so created timestamps differ
		time.Sleep(1 * time.Millisecond)
	}

	acts, err := repo.ListByEngineer(ctx, eid, 2)
	if err != nil {
		t.Fatalf("ListByEngineer error: %v", err)
	}
	if len(acts) != 2 {
		t.Fatalf("expected 2 activities got %d", len(acts))
	}

	acts, err = repo.ListByEngineer(ctx, eid, -10)
	if err != nil {
		t.Fatalf("ListByEngineer error: %v", err)
	}
	if len(acts) != 3 {
		t.Fatalf("expected 3 activities got %d", len(acts))
	}

	// questions
	if _, err := repo.CreateQuestion(ctx, nil); err == nil {
		t.Fatalf("expected error when creating nil question")
	}

	if _, err = repo.CreateQuestion(ctx, &models.Question{EngineerID: eid, Question: "q1"}); err != nil {
		t.Fatalf("CreateQuestion error: %v", err)
	}

	qs, err := repo.ListUnansweredByEngineer(ctx, eid)
	if err != nil {
		t.Fatalf("ListUnansweredByEngineer error: %v", err)
	}
	if len(qs) == 0 {
		t.Fatalf("expected at least 1 unanswered question")
	}
}

func TestJobCreateAndUpdate(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()
	ctx := context.Background()

	j := &models.Job{Status: "pending"}
	jid, err := repo.CreateJob(ctx, j)
	if err != nil {
		t.Fatalf("CreateJob error: %v", err)
	}
	if jid == 0 {
		t.Fatalf("expected job id > 0")
	}

	if err := repo.UpdateStatus(ctx, jid, "done"); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}

	if _, err := repo.CreateJob(ctx, nil); err == nil {
		t.Fatalf("expected error when creating nil job")
	}
}
