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
	for _, want := range []string{"Criar sprint", "/projects/PLAT/sprints", "name=\"start_date\"", "name=\"end_date\"", "14 dias", "Sprint one"} {
		if !strings.Contains(html.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	if !strings.Contains(html.String(), "/projects/PLAT/sprints/start") || !strings.Contains(html.String(), "Iniciar sprint") {
		t.Fatal("sprint não iniciada deve mostrar ação de início")
	}
	if got := boardStatuses(data); len(got) != 5 || got[0] != "backlog" {
		t.Fatal("preparação deve mostrar backlog")
	}
	data.SprintStarted = true
	if got := boardStatuses(data); len(got) != 4 || got[0] != "todo" {
		t.Fatal("sprint iniciada deve ocultar backlog")
	}
}

func TestIssueListShowsSprintProgressBeforeBacklog(t *testing.T) {
	data := ProjectData{
		Project: database.Project{Key: "PLAT"},
		Sprints: []database.Sprint{{ID: 7, Name: "Sprint atual"}},
		Issues: []database.ListProjectIssuesRow{
			{ProjectKey: "PLAT", Number: 1, Title: "Concluída", Status: "done", SprintID: pgtype.Int8{Int64: 7, Valid: true}},
			{ProjectKey: "PLAT", Number: 2, Title: "Em andamento", Status: "todo", SprintID: pgtype.Int8{Int64: 7, Valid: true}},
			{ProjectKey: "PLAT", Number: 3, Title: "Pendente", Status: "backlog"},
		},
	}
	var html strings.Builder
	if err := IssueList(data).Render(context.Background(), &html); err != nil {
		t.Fatal(err)
	}
	output := html.String()
	if !strings.Contains(output, "value=\"50\"") || !strings.Contains(output, "50%") {
		t.Fatal("sprint deve mostrar 50% de progresso")
	}
	if strings.Index(output, "Sprint atual") > strings.Index(output, "Backlog") {
		t.Fatal("sprint deve aparecer antes do backlog")
	}
}
