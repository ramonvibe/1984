package server

import (
 "errors"
 "net/http"
 "strings"
 "time"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/ramon/trackline/internal/calendar"
 "github.com/ramon/trackline/internal/database"
 "github.com/ramon/trackline/internal/issue"
 "github.com/ramon/trackline/web/pages"
)
func (a *App) dashboard(w http.ResponseWriter,r *http.Request) error {
 var data pages.DashboardData; var err error; ctx:=r.Context()
 data.Base,err=a.base(r); if err!=nil { return err }
 data.Stats,err=a.Queries.DashboardStats(ctx,User(r).WorkspaceID); if err!=nil { return err }
 data.Recent,err=a.Queries.RecentIssues(ctx,database.RecentIssuesParams{WorkspaceID:User(r).WorkspaceID,Limit:8}); if err!=nil { return err }
 data.Activity,err=a.Queries.ListRecentActivity(ctx,database.ListRecentActivityParams{WorkspaceID:User(r).WorkspaceID,Limit:10}); if err!=nil { return err }
 data.Release,err=a.Queries.CurrentRelease(ctx,User(r).WorkspaceID); if err!=nil && !errors.Is(err,pgx.ErrNoRows) { return err }
 data.Groups,err=a.reports(r,0); if err!=nil { return err }
 for key,rows:=range data.Groups { if len(rows)>5 { data.Groups[key]=rows[:5] } }
 return render(w,r,pages.Dashboard(data))
}
func reportDates(r *http.Request) (pgtype.Date,pgtype.Date,error) {
 from,err:=issue.Date(r.URL.Query().Get("from")); if err!=nil { return from,pgtype.Date{},err }
 to,err:=issue.Date(r.URL.Query().Get("to")); if err!=nil { return from,to,err }
 if from.Valid && to.Valid && from.Time.After(to.Time) { return from,to,errors.New("end date must not precede start date") }
 return from,to,nil
}
func (a *App) reports(r *http.Request,projectID int64) (map[string][]database.ReportGroupsRow,error) {
 from,to,err:=reportDates(r); if err!=nil { return nil,err }
 result:=map[string][]database.ReportGroupsRow{}
 for _,group:=range []string{"member","project","type","release","issue"} {
  rows,err:=a.Queries.ReportGroups(r.Context(),database.ReportGroupsParams{GroupBy:group,WorkspaceID:User(r).WorkspaceID,ProjectID:projectID,DateFrom:from,DateTo:to}); if err!=nil { return nil,err }
  result[group]=rows
 }
 return result,nil
}
func (a *App) timePage(w http.ResponseWriter,r *http.Request) error {
 base,err:=a.base(r); if err!=nil { return err }
 groups,err:=a.reports(r,0); if err!=nil { return err }
 return render(w,r,pages.TimeReports(base,groups,r.URL.Query().Get("from"),r.URL.Query().Get("to")))
}
func (a *App) calendarPage(w http.ResponseWriter,r *http.Request) error {
 base,err:=a.base(r); if err!=nil { return err }
 month,err:=calendar.Parse(r.URL.Query().Get("month")); if err!=nil { return errors.New("invalid calendar month") }
 items,err:=a.calendarItems(r,month,""); if err!=nil { return err }
 return render(w,r,pages.Calendar(base,calendar.Build(month,items,"")))
}
func (a *App) calendarItems(r *http.Request,month time.Time,key string) ([]database.CalendarRangeRow,error) {
 start:=month.AddDate(0,0,-(int(month.Weekday())+6)%7)
 return a.Queries.CalendarRange(r.Context(),database.CalendarRangeParams{WorkspaceID:User(r).WorkspaceID,ProjectKey:key,DateFrom:pgtype.Date{Time:start,Valid:true},DateTo:pgtype.Date{Time:start.AddDate(0,0,42),Valid:true}})
}
func (a *App) search(w http.ResponseWriter,r *http.Request) error {
 query:=strings.TrimSpace(r.URL.Query().Get("q")); if len(query)>100 { return errors.New("search must be at most 100 bytes") }
 if key,number,err:=issue.ParseReference(query); err==nil {
  item,err:=a.Queries.GetIssueByKey(r.Context(),database.GetIssueByKeyParams{WorkspaceID:User(r).WorkspaceID,Key:key,Number:number})
  if err==nil { redirect(w,r,issueURL(item.ProjectKey,item.Number)); return nil }
  if !errors.Is(err,pgx.ErrNoRows) { return err }
 }
 var data pages.SearchData; var err error
 data.Base,err=a.base(r); if err!=nil { return err }; data.Query=query
 if query!="" {
  data.Issues,err=a.Queries.SearchIssues(r.Context(),database.SearchIssuesParams{WorkspaceID:User(r).WorkspaceID,Lower:query}); if err!=nil { return err }
  data.Others,err=a.Queries.SearchProjectsAndReleases(r.Context(),database.SearchProjectsAndReleasesParams{WorkspaceID:User(r).WorkspaceID,Column2:pgtype.Text{String:query,Valid:true}}); if err!=nil { return err }
 }
 return render(w,r,pages.Search(data))
}
func (a *App) settings(w http.ResponseWriter,r *http.Request) error {
 var data pages.SettingsData; var err error
 data.Base,err=a.base(r); if err!=nil { return err }
 data.Users,err=a.Queries.ListUsers(r.Context(),User(r).WorkspaceID); if err!=nil { return err }
 data.Labels,err=a.Queries.ListLabels(r.Context(),User(r).WorkspaceID); if err!=nil { return err }
 data.GitHubEnabled=a.GitHub.Client.Enabled()
 return render(w,r,pages.Settings(data))
}
func (a *App) addUser(w http.ResponseWriter,r *http.Request) error {
 if err:=a.Auth.AddUser(r.Context(),User(r),r.FormValue("name"),r.FormValue("email"),r.FormValue("password"),r.FormValue("role")); err!=nil { return err }
 redirect(w,r,"/settings"); return nil
}
func (a *App) addLabel(w http.ResponseWriter,r *http.Request) error {
 if err:=a.Projects.Label(r.Context(),User(r),r.FormValue("name"),r.FormValue("color")); err!=nil { return err }
 redirect(w,r,"/settings"); return nil
}
