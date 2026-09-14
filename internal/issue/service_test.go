package issue
import (
 "net/url"
 "testing"
 "github.com/ramon/trackline/internal/database"
)
func TestValidation(t *testing.T) {
 for _,value:=range []string{"PLAT-0","PLAT--1","PLAT-2147483648","x-1","<script>-2"} { if _,_,err:=ParseReference(value); err==nil { t.Errorf("accepted %q",value) } }
 key,number,err:=ParseReference(" plat-142 "); if err!=nil||key!="PLAT"||number!=142 { t.Fatal("valid reference rejected") }
 original:=database.Issue{Title:"Keep this",Type:"task",Status:"backlog",Priority:"medium"}
 next,err:=apply(original,url.Values{"status":{"in_progress"}}); if err!=nil || next.Title!=original.Title || next.Status!="in_progress" { t.Fatal("partial update failed") }
 if _,err:=apply(original,url.Values{"status":{"invalid"}}); err==nil { t.Fatal("invalid status accepted") }
 if _,err:=apply(original,url.Values{"start_date":{"2026-02-03"},"due_date":{"2026-02-01"}}); err==nil { t.Fatal("reversed dates accepted") }
 if _,err:=apply(original,url.Values{"estimated_minutes":{"-1"}}); err==nil { t.Fatal("negative estimate accepted") }
}
