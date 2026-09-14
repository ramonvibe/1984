package release
import (
 "strings"
 "testing"
 "github.com/ramon/trackline/internal/database"
)
func TestChangelog(t *testing.T) {
 items:=[]database.ListReleaseIssuesRow{
  {Number:3,ProjectKey:"PLAT",Title:"Do not ship",Type:"bug",Status:"review"},
  {Number:2,ProjectKey:"PLAT",Title:"Fix auth",Type:"bug",Status:"done"},
  {Number:1,ProjectKey:"PLAT",Title:"Better **search**",Type:"feature",Status:"done"},
  {Number:4,ProjectKey:"PLAT",Title:"No longer needed",Type:"task",Status:"canceled"},
 }
 got:=Changelog(items)
 expected:="## ✨ Features\n\n- Better \\*\\*search\\*\\* (PLAT-1)\n\n## 🐛 Bug Fixes\n\n- Fix auth (PLAT-2)"
 if got!=expected { t.Fatalf("changelog mismatch:\n%s",got) }
 if strings.Contains(got,"Do not ship") || strings.Contains(got,"No longer") { t.Fatal("uncompleted issues must not be published") }
 items[0],items[3]=items[3],items[0]
 if Changelog(items)!=got { t.Fatal("changelog must be deterministic") }
}
