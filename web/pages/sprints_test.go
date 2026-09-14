package pages

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ramon/trackline/internal/database"
	"strings"
	"testing"
	"time"
)

func TestSprintToolbar(t *testing.T) {
	first := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	data := ProjectData{Project: database.Project{Key: "PLAT"}, SprintID: 7, Sprints: []database.Sprint{{ID: 7, Name: "Sprint one", StartDate: pgtype.Date{Time: first, Valid: true}, EndDate: pgtype.Date{Time: first.AddDate(0, 0, 13), Valid: true}}}}
	var html strings.Builder
	if err := SprintToolbar(data).Render(context.Background(), &html); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Create new sprint", "/projects/PLAT/sprints", "name=\"start_date\"", "name=\"end_date\"", "14 days", "Sprint one"} {
		if !strings.Contains(html.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
}
