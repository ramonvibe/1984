package project

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ramon/trackline/internal/activity"
	"github.com/ramon/trackline/internal/auth"
	"github.com/ramon/trackline/internal/database"
	"github.com/ramon/trackline/internal/validate"
)

func sprintDates(start, end string) (pgtype.Date, pgtype.Date, error) {
	first, err := time.Parse("2006-01-02", start)
	if err != nil || first.Year() < 1 {
		return pgtype.Date{}, pgtype.Date{}, errors.New("digite uma data inicial válida")
	}
	last, err := time.Parse("2006-01-02", end)
	if err != nil || last.Year() < 1 {
		return pgtype.Date{}, pgtype.Date{}, errors.New("digite uma data final válida")
	}
	if last.Before(first) {
		return pgtype.Date{}, pgtype.Date{}, errors.New("a data final não pode ser anterior à inicial")
	}
	return pgtype.Date{Time: first, Valid: true}, pgtype.Date{Time: last, Valid: true}, nil
}

func (s Service) CreateSprint(ctx context.Context, user auth.User, projectID int64, name, start, end string) (database.Sprint, error) {
	var result database.Sprint
	first, last, err := sprintDates(start, end)
	if err != nil {
		return result, err
	}
	if strings.TrimSpace(name) == "" {
		name = "Sprint " + start
	}
	name, err = validate.Required("name", name, 120)
	if err != nil {
		return result, err
	}
	var createsRelease bool
	if err = s.Pool.QueryRow(ctx, "SELECT sprints_are_releases FROM workspaces WHERE id=$1", user.WorkspaceID).Scan(&createsRelease); err != nil {
		return result, err
	}
	err = database.Transaction(ctx, s.Pool, func(q *database.Queries) error {
		project, err := q.GetProject(ctx, database.GetProjectParams{ID: projectID, WorkspaceID: user.WorkspaceID})
		if err != nil {
			return err
		}
		if project.Status != "active" {
			return errors.New("o projeto está arquivado")
		}
		result, err = q.CreateSprint(ctx, database.CreateSprintParams{ProjectID: projectID, Name: name, StartDate: first, EndDate: last})
		if err != nil {
			return err
		}
		if createsRelease {
			if _, err = q.CreateRelease(ctx, database.CreateReleaseParams{ProjectID: projectID, Version: name, Name: name, Status: "planning", Description: "", TargetDate: pgtype.Date{}}); err != nil {
				return err
			}
		}
		return activity.Record(ctx, q, user.WorkspaceID, projectID, 0, user.ID, "sprint.created", map[string]string{"title": name})
	})
	return result, err
}

func (s Service) SetIssueSprint(ctx context.Context, user auth.User, issueID, sprintID int64) error {
	if sprintID < 0 {
		return errors.New("sprint inválida")
	}
	return database.Transaction(ctx, s.Pool, func(q *database.Queries) error {
		item, err := q.LockIssue(ctx, database.LockIssueParams{ID: issueID, WorkspaceID: user.WorkspaceID})
		if err != nil {
			return err
		}
		name := "Sem sprint"
		if sprintID > 0 {
			sprint, err := q.GetProjectSprint(ctx, database.GetProjectSprintParams{ID: sprintID, ProjectID: item.ProjectID})
			if err != nil {
				return err
			}
			name = sprint.Name
		}
		if item.SprintID == database.ID(sprintID) {
			return nil
		}
		if err = q.SetIssueSprint(ctx, database.SetIssueSprintParams{ID: issueID, SprintID: database.ID(sprintID)}); err != nil {
			return err
		}
		return activity.Record(ctx, q, user.WorkspaceID, item.ProjectID, issueID, user.ID, "issue.sprint_changed", map[string]string{"to": name})
	})
}

func (s Service) StartSprint(ctx context.Context, user auth.User, projectID, sprintID int64) error {
	if sprintID < 1 {
		return errors.New("sprint inválida")
	}
	return database.Transaction(ctx, s.Pool, func(q *database.Queries) error {
		project, err := q.GetProject(ctx, database.GetProjectParams{ID: projectID, WorkspaceID: user.WorkspaceID})
		if err != nil {
			return err
		}
		if project.Status != "active" {
			return errors.New("o projeto está arquivado")
		}
		sprint, err := q.GetProjectSprint(ctx, database.GetProjectSprintParams{ID: sprintID, ProjectID: projectID})
		if err != nil {
			return err
		}
		if sprint.StartedAt.Valid {
			return errors.New("a sprint já foi iniciada")
		}
		ids, err := q.AssignTodoToSprint(ctx, database.AssignTodoToSprintParams{ProjectID: projectID, SprintID: database.ID(sprintID)})
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err = activity.Record(ctx, q, user.WorkspaceID, projectID, id, user.ID, "issue.sprint_changed", map[string]string{"to": sprint.Name}); err != nil {
				return err
			}
		}
		return q.MarkSprintStarted(ctx, database.MarkSprintStartedParams{ID: sprintID, ProjectID: projectID})
	})
}

func (s Service) PrepareSprint(ctx context.Context, user auth.User, projectID, sprintID int64) error {
	if sprintID < 1 {
		return errors.New("sprint inválida")
	}
	return database.Transaction(ctx, s.Pool, func(q *database.Queries) error {
		project, err := q.GetProject(ctx, database.GetProjectParams{ID: projectID, WorkspaceID: user.WorkspaceID})
		if err != nil {
			return err
		}
		if project.Status != "active" {
			return errors.New("o projeto está arquivado")
		}
		sprint, err := q.GetProjectSprint(ctx, database.GetProjectSprintParams{ID: sprintID, ProjectID: projectID})
		if err != nil {
			return err
		}
		if sprint.StartedAt.Valid {
			return nil
		}
		return q.PrepareSprintIssues(ctx, projectID)
	})
}
