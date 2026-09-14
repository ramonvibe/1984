package server

import (
 "errors"
 "net/http"
 "strconv"
 "time"
 "github.com/ramon/trackline/internal/database"
 "github.com/ramon/trackline/internal/issue"
 "github.com/ramon/trackline/web/pages"
)

func (a *App) findIssue(r *http.Request) (database.GetIssueByKeyRow,error) {
 key,number,err:=issue.ParseReference(r.PathValue("ref")); if err!=nil { return database.GetIssueByKeyRow{},err }
 return a.Queries.GetIssueByKey(r.Context(),database.GetIssueByKeyParams{WorkspaceID:User(r).WorkspaceID,Key:key,Number:number})
}
func (a *App) issueData(r *http.Request) (pages.IssueData,error) {
 var data pages.IssueData; var err error; ctx:=r.Context()
 data.Base,err=a.base(r); if err!=nil { return data,err }
 data.Item,err=a.findIssue(r); if err!=nil { return data,err }
 data.Sprints,err=a.Queries.ListProjectSprints(ctx,data.Item.ProjectID); if err!=nil { return data,err }
 data.Users,err=a.Queries.ListUsers(ctx,User(r).WorkspaceID); if err!=nil { return data,err }
 data.Releases,err=a.Queries.ListProjectReleaseOptions(ctx,data.Item.ProjectID); if err!=nil { return data,err }
 data.Labels,err=a.Queries.ListLabels(ctx,User(r).WorkspaceID); if err!=nil { return data,err }
 data.SelectedLabels,err=a.Queries.ListIssueLabels(ctx,data.Item.ID); if err!=nil { return data,err }
 data.Comments,err=a.Queries.ListComments(ctx,data.Item.ID); if err!=nil { return data,err }
 data.Activity,err=a.Queries.ListIssueActivity(ctx,database.ID(data.Item.ID)); if err!=nil { return data,err }
 data.Time,err=a.Queries.ListIssueTime(ctx,data.Item.ID); if err!=nil { return data,err }
 data.TimeSummary,err=a.Queries.IssueTimeSummary(ctx,data.Item.ID); if err!=nil { return data,err }
 for _,member:=range data.TimeSummary { data.Spent+=member.Seconds }
 data.Commits,err=a.Queries.ListIssueCommits(ctx,data.Item.ID); if err!=nil { return data,err }
 data.PRs,err=a.Queries.ListIssuePullRequests(ctx,data.Item.ID)
 return data,err
}
func (a *App) issuePage(w http.ResponseWriter,r *http.Request) error {
 data,err:=a.issueData(r); if err!=nil { return err }
 if r.Header.Get("HX-Request")=="true" { return render(w,r,pages.IssueDetail(data)) }
 return render(w,r,pages.Issue(data))
}
func (a *App) issueResult(w http.ResponseWriter,r *http.Request) error {
 if r.Header.Get("HX-Request")=="true" { return a.issuePage(w,r) }
 redirect(w,r,"/issues/"+r.PathValue("ref")); return nil
}
func (a *App) newIssue(w http.ResponseWriter,r *http.Request) error {
 var data pages.NewIssueData; var err error
 data.Base,err=a.base(r); if err!=nil { return err }
 data.Project,err=a.findProject(r); if err!=nil { return err }
 data.Users,err=a.Queries.ListUsers(r.Context(),User(r).WorkspaceID); if err!=nil { return err }
 data.Releases,err=a.Queries.ListProjectReleaseOptions(r.Context(),data.Project.ID); if err!=nil { return err }
 data.Labels,err=a.Queries.ListLabels(r.Context(),User(r).WorkspaceID); if err!=nil { return err }
 return render(w,r,pages.NewIssue(data))
}
func (a *App) createIssue(w http.ResponseWriter,r *http.Request) error {
 project,err:=a.findProject(r); if err!=nil { return err }
 item,err:=a.Issues.Create(r.Context(),User(r),project.ID,r.PostForm); if err!=nil { return err }
 redirect(w,r,issueURL(project.Key,item.Number)); return nil
}
func (a *App) updateIssue(w http.ResponseWriter,r *http.Request) error {
 item,err:=a.findIssue(r); if err!=nil { return err }
 if err=a.Issues.Update(r.Context(),User(r),item.ID,r.PostForm); err!=nil { return err }
 return a.issueResult(w,r)
}
func (a *App) moveIssue(w http.ResponseWriter,r *http.Request) error {
 item,err:=a.findIssue(r); if err!=nil { return err }
 for key:=range r.PostForm { if key!="status" && key!="csrf" { return errors.New("only status can be changed by this action") } }
 if r.PostForm.Get("status")=="" { return errors.New("status is required") }
 if err=a.Issues.Update(r.Context(),User(r),item.ID,r.PostForm); err!=nil { return err }
 w.WriteHeader(204); return nil
}
func (a *App) createComment(w http.ResponseWriter,r *http.Request) error {
 item,err:=a.findIssue(r); if err!=nil { return err }
 if err=a.Comments.Create(r.Context(),User(r),item.ID,r.FormValue("body")); err!=nil { return err }
 return a.issueResult(w,r)
}
func (a *App) trackTime(w http.ResponseWriter,r *http.Request) error {
 item,err:=a.findIssue(r); if err!=nil { return err }
 var minutes int64; var started time.Time
 if r.FormValue("action")=="manual" {
  minutes,err=strconv.ParseInt(r.FormValue("minutes"),10,64); if err!=nil { return errors.New("enter a valid number of minutes") }
  started,err=time.Parse("2006-01-02",r.FormValue("date")); if err!=nil { return errors.New("enter a valid date") }
 }
 if err=a.Time.Track(r.Context(),User(r),item.ID,r.FormValue("action"),r.FormValue("description"),minutes,started); err!=nil { return err }
 redirect(w,r,issueURL(item.ProjectKey,item.Number)); return nil
}
