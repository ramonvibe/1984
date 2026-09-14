package github

import (
 "context"
 "encoding/json"
 "errors"
 "strings"
 "time"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/jackc/pgx/v5/pgxpool"
 "github.com/ramon/trackline/internal/activity"
 "github.com/ramon/trackline/internal/auth"
 "github.com/ramon/trackline/internal/database"
 "github.com/ramon/trackline/internal/issue"
)
type Service struct { Pool *pgxpool.Pool; Queries *database.Queries; Client *Client }
func (s Service) Connect(ctx context.Context,user auth.User,projectID,installationID int64,name string) error {
 if user.Role!="admin" { return auth.ErrForbidden }
 if _,err:=s.Queries.GetProject(ctx,database.GetProjectParams{ID:projectID,WorkspaceID:user.WorkspaceID}); err!=nil { return err }
 installation,repository,err:=s.Client.VerifyRepository(ctx,installationID,strings.TrimSpace(name)); if err!=nil { return err }
 return database.Transaction(ctx,s.Pool,func(q *database.Queries) error {
  installed,err:=q.UpsertGitHubInstallation(ctx,database.UpsertGitHubInstallationParams{WorkspaceID:user.WorkspaceID,InstallationID:installationID,AccountLogin:installation.Account.Login,AccountID:database.ID(installation.Account.ID)}); if err!=nil { return err }
  if installed.WorkspaceID!=user.WorkspaceID { return errors.New("installation belongs to another workspace") }
  _,err=q.LinkGitHubRepository(ctx,database.LinkGitHubRepositoryParams{ProjectID:projectID,InstallationID:installed.ID,RepositoryID:repository.ID,Owner:repository.Owner.Login,Name:repository.Name,FullName:repository.FullName,DefaultBranch:repository.DefaultBranch}); return err
 })
}

type payload struct {
 Action string `json:"action"`
 Installation Installation `json:"installation"`
 Repository Repository `json:"repository"`
 Commits []struct{ ID string `json:"id"`; Message string `json:"message"`; URL string `json:"url"`; Timestamp time.Time `json:"timestamp"`; Author struct{Name string `json:"name"`} `json:"author"` } `json:"commits"`
 PullRequest struct{ Number int32 `json:"number"`; Title string `json:"title"`; Body string `json:"body"`; State string `json:"state"`; Merged bool `json:"merged"`; URL string `json:"html_url"`; Head struct{Ref string `json:"ref"`} `json:"head"` } `json:"pull_request"`
 Release PublishedRelease `json:"release"`
}

func (s Service) Webhook(ctx context.Context,delivery,event string,body []byte) error {
 if delivery=="" || len(delivery)>128 { return errors.New("invalid webhook delivery ID") }
 var data payload
 if err:=json.Unmarshal(body,&data); err!=nil { return errors.New("invalid webhook JSON") }
 return database.Transaction(ctx,s.Pool,func(q *database.Queries) error {
  rows,err:=q.ClaimGitHubDelivery(ctx,database.ClaimGitHubDeliveryParams{DeliveryID:delivery,Event:event}); if err!=nil || rows==0 { return err }
  if event=="installation" {
   if data.Action=="deleted" || data.Action=="suspend" { return q.DeleteGitHubInstallation(ctx,data.Installation.ID) }
   // Installation ownership is claimed by an authenticated workspace admin, not by a webhook.
   return nil
  }
  if event!="push" && event!="pull_request" && event!="release" { return nil }
  repository,err:=q.GetGitHubRepositoryByExternalID(ctx,data.Repository.ID)
  if errors.Is(err,pgx.ErrNoRows) { return nil }; if err!=nil { return err }
  linked,err:=q.GetGitHubRepositoryByProject(ctx,repository.ProjectID); if err!=nil { return err }
  if linked.GithubInstallationID!=data.Installation.ID { return errors.New("webhook installation does not match repository") }
  link:=func(text string,work func(database.Issue) error) error {
   for _,reference:=range References(text) {
    key,number,err:=issue.ParseReference(reference); if err!=nil || key!=repository.ProjectKey { continue }
    item,err:=q.FindIssueReference(ctx,database.FindIssueReferenceParams{WorkspaceID:repository.WorkspaceID,Upper:key,Number:number})
    if errors.Is(err,pgx.ErrNoRows) { continue }; if err!=nil { return err }
    if err=work(item); err!=nil { return err }
   }
   return nil
  }
  switch event {
  case "push":
   for _,commit:=range data.Commits {
    if err=link(commit.Message,func(item database.Issue) error {
     changed,err:=q.UpsertGitHubCommit(ctx,database.UpsertGitHubCommitParams{RepositoryID:repository.ID,IssueID:item.ID,Sha:commit.ID,Message:commit.Message,Url:commit.URL,AuthorName:commit.Author.Name,AuthoredAt:database.Timestamp(commit.Timestamp)})
     if err!=nil || changed==0 { return err }
     return activity.Record(ctx,q,repository.WorkspaceID,item.ProjectID,item.ID,0,"github.commit_linked",map[string]string{"title":commit.ID})
    }); err!=nil { return err }
   }
  case "pull_request":
   pr:=data.PullRequest
   if pr.Number<=0 { return errors.New("invalid pull request number") }
   return link(pr.Title+" "+pr.Body+" "+pr.Head.Ref,func(item database.Issue) error {
    changed,err:=q.UpsertGitHubPullRequest(ctx,database.UpsertGitHubPullRequestParams{RepositoryID:repository.ID,IssueID:item.ID,Number:pr.Number,Title:pr.Title,State:pr.State,Merged:pr.Merged,Url:pr.URL})
    if err!=nil || changed==0 { return err }
    kind:="github.pr_linked"; if pr.Merged { kind="github.pr_merged" }
    return activity.Record(ctx,q,repository.WorkspaceID,item.ProjectID,item.ID,0,kind,map[string]string{"title":pr.Title})
   })
  case "release":
   if data.Action=="deleted" { return q.UpdateGitHubReleaseFromWebhook(ctx,database.UpdateGitHubReleaseFromWebhookParams{GithubReleaseID:database.ID(data.Release.ID),GithubUrl:pgtype.Text{},Status:"planning",ProjectID:repository.ProjectID}) }
   if !data.Release.Draft { return q.UpdateGitHubReleaseFromWebhook(ctx,database.UpdateGitHubReleaseFromWebhookParams{GithubReleaseID:database.ID(data.Release.ID),GithubUrl:pgtype.Text{String:data.Release.HTMLURL,Valid:true},Status:"released",ProjectID:repository.ProjectID}) }
  }
  return nil
 })
}
