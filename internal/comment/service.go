package comment

import (
 "context"
 "github.com/jackc/pgx/v5/pgxpool"
 "github.com/ramon/trackline/internal/activity"
 "github.com/ramon/trackline/internal/auth"
 "github.com/ramon/trackline/internal/database"
 "github.com/ramon/trackline/internal/validate"
)
type Service struct { Pool *pgxpool.Pool }
func (s Service) Create(ctx context.Context,user auth.User,issueID int64,body string) error {
 body,err:=validate.Required("comment",body,10000); if err!=nil { return err }
 return database.Transaction(ctx,s.Pool,func(q *database.Queries) error {
  item,err:=q.LockIssue(ctx,database.LockIssueParams{ID:issueID,WorkspaceID:user.WorkspaceID}); if err!=nil { return err }
  if _,err=q.CreateComment(ctx,database.CreateCommentParams{IssueID:issueID,UserID:user.ID,Body:body}); err!=nil { return err }
  return activity.Record(ctx,q,user.WorkspaceID,item.ProjectID,item.ID,user.ID,"comment.created",nil)
 })
}
