package project

import (
 "context"
 "errors"
 "regexp"
 "github.com/jackc/pgx/v5/pgxpool"
 "github.com/ramon/trackline/internal/auth"
 "github.com/ramon/trackline/internal/database"
 "github.com/ramon/trackline/internal/validate"
)
type Service struct { Pool *pgxpool.Pool; Queries *database.Queries }
func (s Service) Create(ctx context.Context,user auth.User,name,key,description string) (database.Project,error) {
 var result database.Project
 if user.Role!="admin" { return result,auth.ErrForbidden }
 name,err:=validate.Required("name",name,120); if err!=nil { return result,err }
 key,err=validate.ProjectKey(key); if err!=nil { return result,err }
 if len(description)>20000 { return result,errors.New("a descrição é muito longa") }
 err=database.Transaction(ctx,s.Pool,func(q *database.Queries) error {
  var err error
  result,err=q.CreateProject(ctx,database.CreateProjectParams{WorkspaceID:user.WorkspaceID,Name:name,Key:key,Description:description})
  if err!=nil { return err }
  return q.AddProjectMember(ctx,database.AddProjectMemberParams{ProjectID:result.ID,UserID:user.ID})
 })
 return result,err
}
func (s Service) Update(ctx context.Context,user auth.User,id int64,name,description,status string) error {
 if user.Role!="admin" { return auth.ErrForbidden }
 if _,err:=s.Queries.GetProject(ctx,database.GetProjectParams{ID:id,WorkspaceID:user.WorkspaceID}); err!=nil { return err }
 name,err:=validate.Required("name",name,120); if err!=nil { return err }
 if len(description)>20000 { return errors.New("a descrição é muito longa") }
 if _,err=validate.OneOf("status",status,"active","archived"); err!=nil { return err }
 _,err=s.Queries.UpdateProject(ctx,database.UpdateProjectParams{ID:id,Name:name,Description:description,Status:status}); return err
}
func (s Service) Member(ctx context.Context,user auth.User,projectID,memberID int64,remove bool) error {
 if user.Role!="admin" { return auth.ErrForbidden }
 if _,err:=s.Queries.GetProject(ctx,database.GetProjectParams{ID:projectID,WorkspaceID:user.WorkspaceID}); err!=nil { return err }
 if _,err:=s.Queries.GetWorkspaceUser(ctx,database.GetWorkspaceUserParams{ID:memberID,WorkspaceID:user.WorkspaceID}); err!=nil { return err }
 if remove { return s.Queries.RemoveProjectMember(ctx,database.RemoveProjectMemberParams{ProjectID:projectID,UserID:memberID}) }
 return s.Queries.AddProjectMember(ctx,database.AddProjectMemberParams{ProjectID:projectID,UserID:memberID})
}
func (s Service) Label(ctx context.Context,user auth.User,name,color string) error {
 name,err:=validate.Required("label",name,40); if err!=nil { return err }
 if !regexp.MustCompile(`^#[0-9a-fA-F]{6}$`).MatchString(color) { return errors.New("cor de etiqueta inválida") }
 _,err=s.Queries.CreateLabel(ctx,database.CreateLabelParams{WorkspaceID:user.WorkspaceID,Name:name,Color:color}); return err
}
