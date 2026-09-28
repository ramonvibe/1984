package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/ramon/trackline/internal/database"
)

func (a *App) createSprint(w http.ResponseWriter, r *http.Request) error {
	project, err := a.findProject(r)
	if err != nil {
		return err
	}
	sprint, err := a.Projects.CreateSprint(r.Context(), User(r), project.ID, r.FormValue("name"), r.FormValue("start_date"), r.FormValue("end_date"))
	if err != nil {
		return err
	}
	if err = a.Projects.PrepareSprint(r.Context(), User(r), project.ID, sprint.ID); err != nil {
		return err
	}
	redirect(w, r, "/projects/"+project.Key+"/board?sprint="+strconv.FormatInt(sprint.ID, 10))
	return nil
}

func (a *App) startSprint(w http.ResponseWriter, r *http.Request) error {
	project, err := a.findProject(r)
	if err != nil {
		return err
	}
	sprintID, err := strconv.ParseInt(r.FormValue("sprint"), 10, 64)
	if err != nil || sprintID < 1 {
		return errors.New("selecione uma sprint para iniciar")
	}
	if err = a.Projects.StartSprint(r.Context(), User(r), project.ID, sprintID); err != nil {
		return err
	}
	sprint, err := a.Queries.GetProjectSprint(r.Context(), database.GetProjectSprintParams{ID: sprintID, ProjectID: project.ID})
	if err != nil {
		return err
	}
	tx, err := a.Pool.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	var creates bool
	if err = tx.QueryRow(r.Context(), "SELECT sprints_are_releases FROM workspaces WHERE id=$1", User(r).WorkspaceID).Scan(&creates); err != nil {
		return err
	}
	if creates {
		if _, err = tx.Exec(r.Context(), "INSERT INTO releases(project_id,version,name,status,target_date) VALUES($1,$2,$2,'planning',$3) ON CONFLICT(project_id,version) DO NOTHING", project.ID, sprint.Name, sprint.EndDate); err != nil {
			return err
		}
	}
	var releaseID int64
	err = tx.QueryRow(r.Context(), "SELECT id FROM releases WHERE project_id=$1 AND version=$2", project.ID, sprint.Name).Scan(&releaseID)
	if err == nil {
		if _, err = tx.Exec(r.Context(), "UPDATE releases SET status='planning',updated_at=now() WHERE project_id=$1 AND status='active' AND id<>$2", project.ID, releaseID); err != nil {
			return err
		}
		if _, err = tx.Exec(r.Context(), "UPDATE releases SET status='active',updated_at=now() WHERE id=$1", releaseID); err != nil {
			return err
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	redirect(w, r, "/projects/"+project.Key+"/board?sprint="+strconv.FormatInt(sprintID, 10))
	return nil
}

func (a *App) selectSprint(w http.ResponseWriter, r *http.Request) error {
	project, err := a.findProject(r)
	if err != nil {
		return err
	}
	value := r.FormValue("sprint")
	sprintID, err := strconv.ParseInt(value, 10, 64)
	if err != nil || sprintID < -1 {
		return errors.New("sprint inválida")
	}
	if sprintID > 0 {
		if err = a.Projects.PrepareSprint(r.Context(), User(r), project.ID, sprintID); err != nil {
			return err
		}
	}
	redirect(w, r, "/projects/"+project.Key+"/board?sprint="+strconv.FormatInt(sprintID, 10))
	return nil
}

func (a *App) setIssueSprint(w http.ResponseWriter, r *http.Request) error {
	item, err := a.findIssue(r)
	if err != nil {
		return err
	}
	var id int64
	if value := r.FormValue("sprint_id"); value != "" {
		id, err = strconv.ParseInt(value, 10, 64)
		if err != nil || id < 1 {
			return errors.New("sprint inválida")
		}
	}
	if err = a.Projects.SetIssueSprint(r.Context(), User(r), item.ID, id); err != nil {
		return err
	}
	return a.issueResult(w, r)
}
