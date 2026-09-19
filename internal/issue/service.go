package issue

import (
 "context"
 "errors"
 "fmt"
 "net/url"
 "strconv"
 "strings"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/jackc/pgx/v5/pgxpool"
 "github.com/ramon/trackline/internal/activity"
 "github.com/ramon/trackline/internal/auth"
 "github.com/ramon/trackline/internal/database"
 "github.com/ramon/trackline/internal/validate"
)
var Statuses=[]string{"backlog","todo","in_progress","review","done","canceled"}
var Types=[]string{"bug","feature","improvement","task"}
var Priorities=[]string{"low","medium","high","urgent"}
type Service struct { Pool *pgxpool.Pool; Queries *database.Queries }

func Reference(key string,number int32) string { return fmt.Sprintf("%s-%d",key,number) }
func ParseReference(value string) (string,int32,error) {
 parts:=strings.Split(strings.ToUpper(strings.TrimSpace(value)),"-")
 if len(parts)!=2 { return "",0,errors.New("código de tarefa inválido") }
 key,err:=validate.ProjectKey(parts[0]); if err!=nil { return "",0,err }
 number,err:=strconv.ParseInt(parts[1],10,32); if err!=nil || number<1 { return "",0,errors.New("número de tarefa inválido") }
 return key,int32(number),nil
}
func Date(value string) (pgtype.Date,error) {
 date,err:=validate.Date(value); if err!=nil || date==nil { return pgtype.Date{},err }
 return pgtype.Date{Time:*date,Valid:true},nil
}
func optionalID(value string) (pgtype.Int8,error) {
 id,err:=validate.OptionalID(value); if err!=nil || id==nil { return pgtype.Int8{},err }; return database.ID(*id),nil
}
func apply(current database.Issue,form url.Values) (database.Issue,error) {
 var err error
 if form.Has("title") { current.Title,err=validate.Required("title",form.Get("title"),240); if err!=nil { return current,err } }
 if form.Has("description") { current.Description=form.Get("description"); if len(current.Description)>50000 { return current,errors.New("a descrição é muito longa") } }
 for _,field:=range []struct{name string; target *string; values []string}{{"type",&current.Type,Types},{"status",&current.Status,Statuses},{"priority",&current.Priority,Priorities}} {
  if form.Has(field.name) { *field.target,err=validate.OneOf(field.name,form.Get(field.name),field.values...); if err!=nil { return current,err } }
 }
 if form.Has("assignee_id") { current.AssigneeID,err=optionalID(form.Get("assignee_id")); if err!=nil { return current,err } }
 if form.Has("release_id") { current.ReleaseID,err=optionalID(form.Get("release_id")); if err!=nil { return current,err } }
 if form.Has("estimated_minutes") {
  value:=form.Get("estimated_minutes"); if value=="" { value="0" }
  minutes,err:=strconv.ParseInt(value,10,32); if err!=nil || minutes<0 || minutes>525600 { return current,errors.New("a estimativa deve estar entre 0 e 525600 minutos") }; current.EstimatedMinutes=int32(minutes)
 }
 if form.Has("start_date") { current.StartDate,err=Date(form.Get("start_date")); if err!=nil { return current,err } }
 if form.Has("due_date") { current.DueDate,err=Date(form.Get("due_date")); if err!=nil { return current,err } }
 if current.StartDate.Valid && current.DueDate.Valid && current.DueDate.Time.Before(current.StartDate.Time) { return current,errors.New("a data limite não pode ser anterior à data inicial") }
 if current.Title=="" { return current,errors.New("o título é obrigatório") }
 return current,nil
}
func relationships(ctx context.Context,q *database.Queries,user auth.User,current database.Issue,form url.Values) error {
 if current.AssigneeID.Valid {
  if _,err:=q.GetWorkspaceUser(ctx,database.GetWorkspaceUserParams{ID:current.AssigneeID.Int64,WorkspaceID:user.WorkspaceID}); err!=nil { return errors.New("a pessoa responsável deve pertencer a este espaço de trabalho") }
 }
 if current.ReleaseID.Valid {
  release,err:=q.GetRelease(ctx,database.GetReleaseParams{ID:current.ReleaseID.Int64,WorkspaceID:user.WorkspaceID})
  if err!=nil || release.ProjectID!=current.ProjectID { return errors.New("a versão deve pertencer a este projeto") }
 }
 if form.Has("labels_present") {
  if len(form["label_id"])>20 { return errors.New("escolha no máximo 20 etiquetas") }
  if err:=q.SetIssueLabels(ctx,current.ID); err!=nil { return err }
  for _,value:=range form["label_id"] {
   id,err:=optionalID(value); if err!=nil || !id.Valid { return errors.New("etiqueta inválida") }
   if _,err=q.GetWorkspaceLabel(ctx,database.GetWorkspaceLabelParams{ID:id.Int64,WorkspaceID:user.WorkspaceID}); err!=nil { return errors.New("a etiqueta deve pertencer a este espaço de trabalho") }
   if err=q.AddIssueLabel(ctx,database.AddIssueLabelParams{IssueID:current.ID,LabelID:id.Int64}); err!=nil { return err }
  }
 }
 return nil
}
func (s Service) Create(ctx context.Context,user auth.User,projectID int64,form url.Values) (database.Issue,error) {
 current,err:=apply(database.Issue{ProjectID:projectID,ReporterID:user.ID,Status:"backlog",Type:"task",Priority:"medium"},form); if err!=nil { return current,err }
 err=database.Transaction(ctx,s.Pool,func(q *database.Queries) error {
  project,err:=q.GetProject(ctx,database.GetProjectParams{ID:projectID,WorkspaceID:user.WorkspaceID}); if err!=nil { return err }; if project.Status=="archived" { return errors.New("o projeto está arquivado") }
  number,err:=q.ReserveIssueNumber(ctx,projectID); if err!=nil { return err }
  current,err=q.CreateIssue(ctx,database.CreateIssueParams{ProjectID:projectID,Number:number,Title:current.Title,Description:current.Description,Type:current.Type,Status:current.Status,Priority:current.Priority,AssigneeID:current.AssigneeID,ReporterID:user.ID,EstimatedMinutes:current.EstimatedMinutes,StartDate:current.StartDate,DueDate:current.DueDate,ReleaseID:current.ReleaseID}); if err!=nil { return err }
  if err=relationships(ctx,q,user,current,form); err!=nil { return err }
  return activity.Record(ctx,q,user.WorkspaceID,projectID,current.ID,user.ID,"issue.created",nil)
 })
 return current,err
}
func (s Service) Update(ctx context.Context,user auth.User,id int64,form url.Values) error {
 return database.Transaction(ctx,s.Pool,func(q *database.Queries) error {
  previous,err:=q.LockIssue(ctx,database.LockIssueParams{ID:id,WorkspaceID:user.WorkspaceID}); if err!=nil { return err }
  current,err:=apply(previous,form); if err!=nil { return err }
  if err=relationships(ctx,q,user,current,form); err!=nil { return err }
  _,err=q.UpdateIssue(ctx,database.UpdateIssueParams{ID:id,Title:current.Title,Description:current.Description,Type:current.Type,Status:current.Status,Priority:current.Priority,AssigneeID:current.AssigneeID,EstimatedMinutes:current.EstimatedMinutes,StartDate:current.StartDate,DueDate:current.DueDate,ReleaseID:current.ReleaseID}); if err!=nil { return err }
  recorded:=false
  for _,change:=range []struct{kind,from,to string}{{"issue.status_changed",previous.Status,current.Status},{"issue.priority_changed",previous.Priority,current.Priority},{"issue.assigned",fmt.Sprint(previous.AssigneeID),fmt.Sprint(current.AssigneeID)}} {
   if change.from==change.to { continue }
   data:=map[string]string{"from":change.from,"to":change.to}
   if change.kind=="issue.assigned" {
    data["to"]="Unassigned"
    if current.AssigneeID.Valid { member,err:=q.GetWorkspaceUser(ctx,database.GetWorkspaceUserParams{ID:current.AssigneeID.Int64,WorkspaceID:user.WorkspaceID}); if err!=nil { return err }; data["to"]=member.Name }
   }
   if err=activity.Record(ctx,q,user.WorkspaceID,current.ProjectID,id,user.ID,change.kind,data); err!=nil { return err }; recorded=true
  }
  if !recorded || previous.Title!=current.Title || previous.Description!=current.Description || form.Has("labels_present") { return activity.Record(ctx,q,user.WorkspaceID,current.ProjectID,id,user.ID,"issue.updated",nil) }
  return nil
 })
}
