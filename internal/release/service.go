package release

import (
 "context"
 "errors"
 "fmt"
 "net/url"
 "regexp"
 "sort"
 "strings"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/jackc/pgx/v5/pgxpool"
 "github.com/ramon/trackline/internal/activity"
 "github.com/ramon/trackline/internal/auth"
 "github.com/ramon/trackline/internal/database"
 "github.com/ramon/trackline/internal/github"
 "github.com/ramon/trackline/internal/issue"
 "github.com/ramon/trackline/internal/validate"
)
var Statuses=[]string{"planning","active","released","canceled"}
type Service struct { Pool *pgxpool.Pool; Queries *database.Queries; GitHub *github.Client }
var versionPattern=regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$`)
func fields(form url.Values) (database.CreateReleaseParams,error) {
 value:=database.CreateReleaseParams{Version:strings.TrimSpace(form.Get("version")),Name:strings.TrimSpace(form.Get("name")),Description:form.Get("description"),Status:form.Get("status")}
 if !versionPattern.MatchString(value.Version) || strings.Contains(value.Version,"..") || strings.HasSuffix(value.Version,".") || strings.HasSuffix(value.Version,".lock") { return value,errors.New("version must be a valid tag using letters, numbers, dots, hyphens or underscores") }
 if len(value.Name)>120 || len(value.Description)>20000 { return value,errors.New("release name or description is too long") }
 if value.Status=="" { value.Status="planning" }
 _,err:=validate.OneOf("release status",value.Status,Statuses...); if err!=nil { return value,err }
 value.TargetDate,err=issue.Date(form.Get("target_date")); return value,err
}
func (s Service) Create(ctx context.Context,user auth.User,projectID int64,form url.Values) (database.Release,error) {
 var result database.Release
 params,err:=fields(form); if err!=nil { return result,err }; params.ProjectID=projectID
 err=database.Transaction(ctx,s.Pool,func(q *database.Queries) error {
  if _,err:=q.GetProject(ctx,database.GetProjectParams{ID:projectID,WorkspaceID:user.WorkspaceID}); err!=nil { return err }
  var err error; result,err=q.CreateRelease(ctx,params); if err!=nil { return err }
  return activity.Record(ctx,q,user.WorkspaceID,projectID,0,user.ID,"release.created",map[string]string{"title":result.Version})
 }); return result,err
}
func (s Service) Update(ctx context.Context,user auth.User,id int64,form url.Values) error {
 params,err:=fields(form); if err!=nil { return err }
 return database.Transaction(ctx,s.Pool,func(q *database.Queries) error {
  item,err:=q.GetRelease(ctx,database.GetReleaseParams{ID:id,WorkspaceID:user.WorkspaceID}); if err!=nil { return err }
  if item.GithubReleaseID.Valid { return errors.New("published GitHub releases cannot be edited here") }
  if _,err=q.UpdateRelease(ctx,database.UpdateReleaseParams{ID:id,Version:params.Version,Name:params.Name,Description:params.Description,Status:params.Status,TargetDate:params.TargetDate}); err!=nil { return err }
  return activity.Record(ctx,q,user.WorkspaceID,item.ProjectID,0,user.ID,"release.updated",map[string]string{"title":params.Version})
 })
}
// Changelog is a deterministic transformation, independent of storage or optional rewrite integrations.
func Changelog(items []database.ListReleaseIssuesRow) string {
 items=append([]database.ListReleaseIssuesRow(nil),items...)
 sort.Slice(items,func(i,j int) bool { return items[i].Number<items[j].Number })
 var result strings.Builder
 for _,group:=range []struct{kind,title string}{{"feature","✨ Features"},{"improvement","⚡ Improvements"},{"bug","🐛 Bug Fixes"},{"task","🔧 Other Changes"}} {
  var lines []string
  for _,item:=range items {
   if item.Status!="done" || item.Type!=group.kind { continue }
   title:=strings.Join(strings.Fields(item.Title)," ")
   title=strings.NewReplacer("\\","\\\\","[","\\[","]","\\]","*","\\*","_","\\_","`","\\`","<","&lt;",">","&gt;").Replace(title)
   lines=append(lines,fmt.Sprintf("- %s (%s)",title,issue.Reference(item.ProjectKey,item.Number)))
  }
  if len(lines)>0 { result.WriteString("## "+group.title+"\n\n"+strings.Join(lines,"\n")+"\n\n") }
 }
 return strings.TrimSpace(result.String())
}
func (s Service) Save(ctx context.Context,user auth.User,id int64,changelog string,generate bool) error {
 if len(changelog)>100000 { return errors.New("changelog is too long") }
 return database.Transaction(ctx,s.Pool,func(q *database.Queries) error {
  item,err:=q.GetRelease(ctx,database.GetReleaseParams{ID:id,WorkspaceID:user.WorkspaceID}); if err!=nil { return err }
  locked,err:=q.LockRelease(ctx,id); if err!=nil { return err }; if locked.GithubReleaseID.Valid { return errors.New("release is already published") }
  if generate { items,err:=q.ListReleaseIssues(ctx,database.ID(id)); if err!=nil { return err }; changelog=Changelog(items) }
  if _,err=q.SaveReleaseChangelog(ctx,database.SaveReleaseChangelogParams{ID:id,Changelog:changelog}); err!=nil { return err }
  return activity.Record(ctx,q,user.WorkspaceID,item.ProjectID,0,user.ID,"release.changelog_saved",map[string]string{"title":item.Version})
 })
}
func (s Service) Publish(ctx context.Context,user auth.User,id int64) error {
 if user.Role!="admin" { return auth.ErrForbidden }
 if !s.GitHub.Enabled() { return errors.New("GitHub App is not configured") }
 return database.Transaction(ctx,s.Pool,func(q *database.Queries) error {
  scoped,err:=q.GetRelease(ctx,database.GetReleaseParams{ID:id,WorkspaceID:user.WorkspaceID}); if err!=nil { return err }
  item,err:=q.LockRelease(ctx,id); if err!=nil { return err }
  if item.GithubReleaseID.Valid { return nil }
  if item.Status=="canceled" || strings.TrimSpace(item.Changelog)=="" { return errors.New("save a changelog for a non-canceled release before publishing") }
  repository,err:=q.GetGitHubRepositoryByProject(ctx,item.ProjectID); if err!=nil { return errors.New("connect a GitHub repository in project settings first") }
  name:=item.Name; if name=="" { name=item.Version }
  // ponytail: hold one release row during a bounded HTTP call; use an outbox if publishing throughput grows.
  published,err:=s.GitHub.Publish(ctx,repository.GithubInstallationID,repository.FullName,item.Version,name,item.Changelog,item.PublicationKey); if err!=nil { return err }
  if _,err=q.MarkReleasePublished(ctx,database.MarkReleasePublishedParams{ID:id,GithubReleaseID:database.ID(published.ID),GithubUrl:pgtype.Text{String:published.HTMLURL,Valid:true},GithubTag:pgtype.Text{String:published.TagName,Valid:true}}); err!=nil { return err }
  return activity.Record(ctx,q,user.WorkspaceID,scoped.ProjectID,0,user.ID,"release.published",map[string]string{"title":item.Version})
 })
}
