package project

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ramon/trackline/internal/auth"
	"github.com/ramon/trackline/internal/database"
	"github.com/ramon/trackline/internal/issue"
)

// Run with TEST_DATABASE_URL. All test data lives in a temporary schema.
func TestSprintDatabase(t *testing.T) {
	connection := os.Getenv("TEST_DATABASE_URL")
	if connection == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration checks")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, connection)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	schema := pgx.Identifier{fmt.Sprintf("sprint_test_%d", time.Now().UnixNano())}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	config, err := pgxpool.ParseConfig(connection)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err = database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	q := database.New(pool)
	workspace, err := q.CreateWorkspace(ctx, "Sprint test")
	if err != nil {
		t.Fatal(err)
	}
	member, err := q.CreateUser(ctx, database.CreateUserParams{WorkspaceID: workspace.ID, Name: "Alice Smith", Email: "alice@example.test", PasswordHash: "unused", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	user := auth.User{ID: member.ID, WorkspaceID: workspace.ID, Role: "admin"}
	service := Service{Pool: pool, Queries: q}
	project, err := service.Create(ctx, user, "Platform", "PLAT", "")
	if err != nil {
		t.Fatal(err)
	}
	other, err := service.Create(ctx, user, "Other", "OTHER", "")
	if err != nil {
		t.Fatal(err)
	}
	sprint, err := service.CreateSprint(ctx, user, project.ID, "", "2026-09-01", "2026-09-14")
	if err != nil {
		t.Fatal(err)
	}
	if sprint.Name != "Sprint 2026-09-01" {
		t.Fatal("missing default name")
	}
	if _, err = service.CreateSprint(ctx, user, project.ID, "Invalid", "2026-09-14", "2026-09-01"); err == nil {
		t.Fatal("reversed dates accepted")
	}
	foreign, err := service.CreateSprint(ctx, user, other.ID, "Other", "2026-09-01", "2026-09-14")
	if err != nil {
		t.Fatal(err)
	}
	issues := issue.Service{Pool: pool, Queries: q}
	item, err := issues.Create(ctx, user, project.ID, url.Values{"title": {"Test issue"}})
	if err != nil {
		t.Fatal(err)
	}
	checkBoard := func(filter int64, count int) {
		t.Helper()
		rows, err := q.ListBoardIssues(ctx, database.ListBoardIssuesParams{ProjectID: project.ID, SprintFilter: filter})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != count {
			t.Fatalf("filter %d: got %d issues, want %d", filter, len(rows), count)
		}
	}
	checkBoard(0, 1)
	checkBoard(-1, 1)
	checkBoard(sprint.ID, 0)
	if err = service.SetIssueSprint(ctx, user, item.ID, sprint.ID); err != nil {
		t.Fatal(err)
	}
	checkBoard(sprint.ID, 1)
	checkBoard(-1, 0)
	if err = issues.Update(ctx, user, item.ID, url.Values{"status": {"in_progress"}}); err != nil {
		t.Fatal(err)
	}
	checkBoard(sprint.ID, 1)
	if err = service.SetIssueSprint(ctx, user, item.ID, foreign.ID); err == nil {
		t.Fatal("cross-project sprint accepted")
	}
	if err = q.SetIssueSprint(ctx, database.SetIssueSprintParams{ID: item.ID, SprintID: database.ID(foreign.ID)}); err == nil {
		t.Fatal("database must reject cross-project sprint")
	}
	outsider := user
	outsider.WorkspaceID = workspace.ID + 100
	if err = service.SetIssueSprint(ctx, outsider, item.ID, 0); err == nil {
		t.Fatal("cross-workspace mutation accepted")
	}
	checkBoard(sprint.ID, 1)
	if err = service.SetIssueSprint(ctx, user, item.ID, 0); err != nil {
		t.Fatal(err)
	}
	checkBoard(-1, 1)
	checkBoard(sprint.ID, 0)
	events, err := q.ListIssueActivity(ctx, database.ID(item.ID))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, event := range events {
		if event.Kind == "issue.sprint_changed" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("got %d sprint change events, want 2", count)
	}
}
