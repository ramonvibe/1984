package timetracking

import (
 "context"
 "errors"
 "time"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgconn"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/jackc/pgx/v5/pgxpool"
 "github.com/ramon/trackline/internal/activity"
 "github.com/ramon/trackline/internal/auth"
 "github.com/ramon/trackline/internal/database"
)
type Service struct { Pool *pgxpool.Pool }
func (s Service) Track(ctx context.Context,user auth.User,issueID int64,action,description string,minutes int64,started time.Time) error {
 if len(description)>2000 { return errors.New("a descrição do tempo é muito longa") }
 if action=="manual" && (minutes<1 || minutes>1440 || started.IsZero() || started.After(time.Now().UTC().Add(24*time.Hour))) { return errors.New("o registro manual exige de 1 a 1440 minutos e uma data válida") }
 err:=database.Transaction(ctx,s.Pool,func(q *database.Queries) error {
  item,err:=q.LockIssue(ctx,database.LockIssueParams{ID:issueID,WorkspaceID:user.WorkspaceID}); if err!=nil { return err }
  kind:=""
  switch action {
  case "start": _,err=q.StartTimer(ctx,database.StartTimerParams{IssueID:issueID,UserID:user.ID,Description:description}); kind="timer.started"
  case "stop": _,err=q.StopTimer(ctx,database.StopTimerParams{UserID:user.ID,IssueID:issueID}); kind="timer.stopped"
  case "manual": _,err=q.AddTimeEntry(ctx,database.AddTimeEntryParams{IssueID:issueID,UserID:user.ID,StartedAt:database.Timestamp(started),DurationSeconds:pgtype.Int4{Int32:int32(minutes*60),Valid:true},Description:description}); kind="time.added"
  default: return errors.New("ação de cronômetro inválida")
  }
  if err!=nil { return err }
  return activity.Record(ctx,q,user.WorkspaceID,item.ProjectID,item.ID,user.ID,kind,nil)
 })
 var pgErr *pgconn.PgError
 if errors.As(err,&pgErr) && pgErr.ConstraintName=="time_entries_one_active_per_user_uidx" { return errors.New("você já tem um cronômetro ativo; pare-o antes de iniciar outro") }
 if errors.Is(err,pgx.ErrNoRows) { return errors.New("nenhum cronômetro ou tarefa correspondente foi encontrado") }
 return err
}
