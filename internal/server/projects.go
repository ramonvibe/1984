package server

import (
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/ramon/trackline/internal/auth"
	"github.com/ramon/trackline/internal/calendar"
	"github.com/ramon/trackline/internal/database"
	"github.com/ramon/trackline/web/pages"
	"net/http"
	"slices"
	"strconv"
)

func (a *App) findProject(r *http.Request) (database.Project, error) {
	return a.Queries.GetProjectByKey(r.Context(), database.GetProjectByKeyParams{WorkspaceID: User(r).WorkspaceID, Key: r.PathValue("key")})
}
func (a *App) projects(w http.ResponseWriter, r *http.Request) error {
	base, err := a.base(r)
	if err != nil {
		return err
	}
	return render(w, r, pages.Projects(base))
}
func (a *App) createProject(w http.ResponseWriter, r *http.Request) error {
	item, err := a.Projects.Create(r.Context(), User(r), r.FormValue("name"), r.FormValue("key"), r.FormValue("description"))
	if err != nil {
		return err
	}
	redirect(w, r, "/projects/"+item.Key)
	return nil
}
func (a *App) projectPage(w http.ResponseWriter, r *http.Request) error {
	var data pages.ProjectData
	var err error
	ctx := r.Context()
	data.Base, err = a.base(r)
	if err != nil {
		return err
	}
	data.Project, err = a.findProject(r)
	if err != nil {
		return err
	}
	data.Tab = r.PathValue("tab")
	data.Page = 1
	switch data.Tab {
	case "", "issues":
		if value := r.URL.Query().Get("page"); value != "" {
			data.Page, err = strconv.Atoi(value)
			if err != nil || data.Page < 1 || data.Page > 100000 {
				return errors.New("página inválida")
			}
		}
		data.Status = r.URL.Query().Get("status")
		data.Type = r.URL.Query().Get("type")
		data.Issues, err = a.Queries.ListProjectIssues(ctx, database.ListProjectIssuesParams{ProjectID: data.Project.ID, Column2: data.Status, Column3: data.Type, Limit: 500, Offset: 0})
		if err != nil {
			return err
		}
		data.Sprints, err = a.Queries.ListProjectSprints(ctx, data.Project.ID)
		if err != nil {
			return err
		}
		if data.Tab == "" {
			data.Members, err = a.Queries.ListProjectMembers(ctx, data.Project.ID)
		}
	case "board":
		if value := r.URL.Query().Get("sprint"); value != "" {
			data.SprintID, err = strconv.ParseInt(value, 10, 64)
			if err != nil || data.SprintID < -1 {
				return errors.New("sprint inválida")
			}
		}
		if _, present := r.URL.Query()["sprint"]; !present {
			_ = a.Pool.QueryRow(ctx, "SELECT id FROM sprints WHERE project_id=$1 AND started_at IS NOT NULL ORDER BY started_at DESC LIMIT 1", data.Project.ID).Scan(&data.SprintID)
		}
		if data.SprintID > 0 {
			selected, queryErr := a.Queries.GetProjectSprint(ctx, database.GetProjectSprintParams{ID: data.SprintID, ProjectID: data.Project.ID})
			if queryErr != nil {
				return queryErr
			}
			data.SprintStarted = selected.StartedAt.Valid
		}
		data.Sprints, err = a.Queries.ListProjectSprints(ctx, data.Project.ID)
		if err != nil {
			return err
		}
		boardFilter := data.SprintID
		if data.SprintID > 0 && !data.SprintStarted {
			boardFilter = -1
		}
		data.Board, err = a.Queries.ListBoardIssues(ctx, database.ListBoardIssuesParams{ProjectID: data.Project.ID, SprintFilter: boardFilter})
		if data.SprintID > 0 && !data.SprintStarted {
			data.Board = slices.DeleteFunc(data.Board, func(item database.ListBoardIssuesRow) bool { return item.Status == "done" || item.Status == "canceled" })
		}
	case "releases":
		data.Releases, err = a.Queries.ListReleases(ctx, data.Project.ID)
	case "settings":
		if User(r).Role != "admin" {
			return auth.ErrForbidden
		}
		data.Members, err = a.Queries.ListProjectMembers(ctx, data.Project.ID)
		if err != nil {
			return err
		}
		data.Users, err = a.Queries.ListUsers(ctx, User(r).WorkspaceID)
		if err != nil {
			return err
		}
		data.Repository, err = a.Queries.GetGitHubRepositoryByProject(ctx, data.Project.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			err = nil
		}
		data.GitHubEnabled = a.GitHub.Client.Enabled()
		data.GitHubSlug = a.Config.GitHubAppSlug
	case "calendar":
		month, parseErr := calendar.Parse(r.URL.Query().Get("month"))
		if parseErr != nil {
			return errors.New("mês de calendário inválido")
		}
		items, queryErr := a.calendarItems(r, month, data.Project.Key)
		if queryErr != nil {
			return queryErr
		}
		data.Calendar = calendar.Build(month, items, data.Project.Key)
	case "time":
		data.DateFrom = r.URL.Query().Get("from")
		data.DateTo = r.URL.Query().Get("to")
		data.Reports, err = a.reports(r, data.Project.ID)
		if err != nil {
			return err
		}
		from, to, parseErr := reportDates(r)
		if parseErr != nil {
			return parseErr
		}
		data.Estimates, err = a.Queries.ProjectTimeReport(ctx, database.ProjectTimeReportParams{ProjectID: data.Project.ID, Column2: from, Column3: to})
	default:
		return pgx.ErrNoRows
	}
	if err != nil {
		return err
	}
	return render(w, r, pages.Project(data))
}
func (a *App) updateProject(w http.ResponseWriter, r *http.Request) error {
	project, err := a.findProject(r)
	if err != nil {
		return err
	}
	if err = a.Projects.Update(r.Context(), User(r), project.ID, r.FormValue("name"), r.FormValue("description"), r.FormValue("status")); err != nil {
		return err
	}
	redirect(w, r, "/projects/"+project.Key+"/settings")
	return nil
}
func (a *App) projectMember(w http.ResponseWriter, r *http.Request) error {
	project, err := a.findProject(r)
	if err != nil {
		return err
	}
	id, err := parseID(r.FormValue("user_id"))
	if err != nil {
		return err
	}
	if err = a.Projects.Member(r.Context(), User(r), project.ID, id, r.FormValue("remove") == "true"); err != nil {
		return err
	}
	redirect(w, r, "/projects/"+project.Key+"/settings")
	return nil
}
func (a *App) connectGitHub(w http.ResponseWriter, r *http.Request) error {
	project, err := a.findProject(r)
	if err != nil {
		return err
	}
	id, err := parseID(r.FormValue("installation_id"))
	if err != nil {
		return err
	}
	if err = a.GitHub.Connect(r.Context(), User(r), project.ID, id, r.FormValue("repository")); err != nil {
		return err
	}
	redirect(w, r, "/projects/"+project.Key+"/settings")
	return nil
}
