package calendar

import (
 "fmt"
 "time"
 "github.com/ramon/trackline/internal/database"
 "github.com/ramon/trackline/internal/issue"
)
type Event struct { Title, URL, Kind string }
type Day struct { Date time.Time; Events []Event; Current, Today bool }
type Month struct { Title, Previous, Next, Value string; Days []Day }
func Parse(value string) (time.Time,error) {
 if value=="" { now:=time.Now().UTC(); return time.Date(now.Year(),now.Month(),1,0,0,0,0,time.UTC),nil }
 return time.Parse("2006-01",value)
}
func Build(month time.Time,items []database.CalendarRangeRow,projectKey string) Month {
 months:=[]string{"janeiro","fevereiro","março","abril","maio","junho","julho","agosto","setembro","outubro","novembro","dezembro"}
 result:=Month{Title:fmt.Sprintf("%s de %d",months[month.Month()-1],month.Year()),Value:month.Format("2006-01"),Previous:month.AddDate(0,-1,0).Format("2006-01"),Next:month.AddDate(0,1,0).Format("2006-01")}
 start:=month.AddDate(0,0,-(int(month.Weekday())+6)%7)
 events:=map[string][]Event{}
 for _,item:=range items {
  if projectKey!="" && item.ProjectKey!=projectKey { continue }
  if item.ItemKind=="release" {
   key:=item.TargetDate.Time.Format("2006-01-02")
   events[key]=append(events[key],Event{Title:item.Title,URL:fmt.Sprintf("/releases/%d",item.ID),Kind:"Release"})
  } else {
   title:=issue.Reference(item.ProjectKey,item.Number)+" · "+item.Title
   url:="/issues/"+issue.Reference(item.ProjectKey,item.Number)
   if item.StartDate.Valid { key:=item.StartDate.Time.Format("2006-01-02"); events[key]=append(events[key],Event{Title:title,URL:url,Kind:"Start"}) }
   if item.DueDate.Valid { key:=item.DueDate.Time.Format("2006-01-02"); events[key]=append(events[key],Event{Title:title,URL:url,Kind:"Due"}) }
  }
 }
 for day:=0;day<42;day++ {
  date:=start.AddDate(0,0,day); key:=date.Format("2006-01-02")
  result.Days=append(result.Days,Day{Date:date,Events:events[key],Current:date.Month()==month.Month(),Today:key==time.Now().UTC().Format("2006-01-02")})
 }
 return result
}

func KindName(value string) string {
 names:=map[string]string{"Release":"Versão","Start":"Início","Due":"Prazo"}
 if name:=names[value]; name!="" { return name }
 return value
}
