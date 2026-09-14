package components

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ramon/trackline/internal/database"
	"strings"
	"testing"
)

func TestCardFirstName(t *testing.T) {
	for _, tc := range []struct{ full, first string }{
		{"Alice Smith", "Alice"}, {"  João  Silva ", "João"}, {"李 明", "李"}, {"Prince", "Prince"}, {" \t ", "Unassigned"},
	} {
		if got := FirstName(tc.full); got != tc.first {
			t.Errorf("FirstName(%q)=%q", tc.full, got)
		}
	}
	var output strings.Builder
	err := IssueCard(database.ListBoardIssuesRow{AssigneeName: pgtype.Text{String: "Alice Smith", Valid: true}}).Render(context.Background(), &output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), ">Alice</span>") || !strings.Contains(output.String(), "title=\"Alice Smith\"") {
		t.Fatal("card must show first name and retain full name tooltip")
	}
}
