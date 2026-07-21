package db

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/KingBoyAndGirl/HomeVox/backend/internal/floorplan"
	"github.com/jackc/pgx/v5/pgxpool"
)

func mustTestDatabaseURL(t *testing.T) string {
	dsn := os.Getenv("HOMEVOX_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("HOMEVOX_TEST_DATABASE_URL not set; skipping integration test")
	}
	return dsn
}

func TestPostgresRepositoryLifecycle(t *testing.T) {
	dsn := mustTestDatabaseURL(t)

	ctx := context.Background()
	repo, err := NewPostgresRepository(ctx, dsn)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	defer repo.Close()

	if err := repo.InitializeSchema(ctx); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}

	if err := truncateProjectsTable(ctx, dsn); err != nil {
		t.Fatalf("truncate table: %v", err)
	}

	doc := floorplan.ParseResult{Walls: []floorplan.Segment{{X1: 0, Y1: 0, X2: 1, Y2: 0}}}
	docJSON, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal doc: %v", err)
	}

	created, err := repo.Create(ctx, "00000000-0000-4000-8000-000000000001", "Suite 1", "source/one.png", "image/png", 12, docJSON)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("created ID = %s, want explicit server ID", created.ID)
	}
	if created.Revision != 1 {
		t.Fatalf("revision = %d, want 1", created.Revision)
	}
	if created.CreatedAt.Location() != time.UTC {
		t.Fatalf("created_at not UTC: %v", created.CreatedAt.Location())
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("get id = %s, want %s", got.ID, created.ID)
	}

	list, err := repo.List(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}

	updatedDoc := floorplan.ParseResult{Walls: []floorplan.Segment{{X1: 0, Y1: 0, X2: 2, Y2: 0}}}
	updatedDocJSON, err := json.Marshal(updatedDoc)
	if err != nil {
		t.Fatalf("marshal updated doc: %v", err)
	}
	updated, err := repo.Update(ctx, created.ID, 1, "Suite 1+", updatedDocJSON)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Revision != 2 {
		t.Fatalf("updated revision = %d, want 2", updated.Revision)
	}
	if _, err := repo.Update(ctx, created.ID, 1, "bad", updatedDocJSON); err == nil {
		t.Fatal("expected revision conflict")
	}
}

func truncateProjectsTable(ctx context.Context, dsn string) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, `TRUNCATE TABLE projects;`)
	return err
}
