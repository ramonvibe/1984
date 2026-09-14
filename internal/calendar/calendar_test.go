package calendar
import (
 "testing"
 "time"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/ramon/trackline/internal/database"
)
func TestCalendarUsesExistingDates(t *testing.T) {
 month,_:=Parse("2026-02")
 date:=pgtype.Date{Time:month.AddDate(0,0,3),Valid:true}
 data:=Build(month,[]database.CalendarRangeRow{{ID:1,Number:7,ProjectKey:"PLAT",Title:"Fix",ItemKind:"issue",StartDate:date,DueDate:date},{ID:2,ProjectKey:"PLAT",Title:"v1",ItemKind:"release",TargetDate:date}},"PLAT")
 if len(data.Days)!=42 || data.Days[0].Date.Weekday()!=time.Monday || data.Previous!="2026-01" || data.Next!="2026-03" { t.Fatal("invalid month boundaries") }
 total:=0;for _,day:=range data.Days {total+=len(day.Events)}
 if total!=3 { t.Fatal("start, due and release dates must each appear") }
}
