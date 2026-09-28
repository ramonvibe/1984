package server

import (
	"errors"
	"github.com/ramon/trackline/internal/database"
	"github.com/ramon/trackline/internal/issue"
	"github.com/ramon/trackline/web/pages"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (a *App) myTasks(w http.ResponseWriter, r *http.Request) error {
	data := pages.MyTasksData{}
	var err error
	data.Base, err = a.base(r)
	if err != nil {
		return err
	}
	rows, err := a.Pool.Query(r.Context(), `SELECT p.key, i.number, i.title, i.status, i.updated_at FROM issues i JOIN projects p ON p.id=i.project_id WHERE p.workspace_id=$1 AND i.assignee_id=$2 ORDER BY CASE i.status WHEN 'in_progress' THEN 1 WHEN 'review' THEN 2 WHEN 'todo' THEN 3 WHEN 'backlog' THEN 4 WHEN 'done' THEN 5 ELSE 6 END, i.updated_at DESC`, User(r).WorkspaceID, User(r).ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item pages.MyTask
		if err = rows.Scan(&item.ProjectKey, &item.Number, &item.Title, &item.Status, &item.UpdatedAt); err != nil {
			return err
		}
		data.Tasks = append(data.Tasks, item)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return render(w, r, pages.MyTasks(data))
}

func (a *App) findIssue(r *http.Request) (database.GetIssueByKeyRow, error) {
	key, number, err := issue.ParseReference(r.PathValue("ref"))
	if err != nil {
		return database.GetIssueByKeyRow{}, err
	}
	return a.Queries.GetIssueByKey(r.Context(), database.GetIssueByKeyParams{WorkspaceID: User(r).WorkspaceID, Key: key, Number: number})
}
func (a *App) issueData(r *http.Request) (pages.IssueData, error) {
	var data pages.IssueData
	var err error
	ctx := r.Context()
	data.Base, err = a.base(r)
	if err != nil {
		return data, err
	}
	data.Item, err = a.findIssue(r)
	if err != nil {
		return data, err
	}
	data.Sprints, err = a.Queries.ListProjectSprints(ctx, data.Item.ProjectID)
	if err != nil {
		return data, err
	}
	data.Users, err = a.Queries.ListUsers(ctx, User(r).WorkspaceID)
	if err != nil {
		return data, err
	}
	data.Releases, err = a.Queries.ListProjectReleaseOptions(ctx, data.Item.ProjectID)
	if err != nil {
		return data, err
	}
	data.Labels, err = a.Queries.ListLabels(ctx, User(r).WorkspaceID)
	if err != nil {
		return data, err
	}
	data.SelectedLabels, err = a.Queries.ListIssueLabels(ctx, data.Item.ID)
	if err != nil {
		return data, err
	}
	data.Comments, err = a.Queries.ListComments(ctx, data.Item.ID)
	if err != nil {
		return data, err
	}
	data.Activity, err = a.Queries.ListIssueActivity(ctx, database.ID(data.Item.ID))
	if err != nil {
		return data, err
	}
	data.Time, err = a.Queries.ListIssueTime(ctx, data.Item.ID)
	if err != nil {
		return data, err
	}
	data.TimeSummary, err = a.Queries.IssueTimeSummary(ctx, data.Item.ID)
	if err != nil {
		return data, err
	}
	for _, member := range data.TimeSummary {
		data.Spent += member.Seconds
	}
	data.Commits, err = a.Queries.ListIssueCommits(ctx, data.Item.ID)
	if err != nil {
		return data, err
	}
	data.PRs, err = a.Queries.ListIssuePullRequests(ctx, data.Item.ID)
	if err != nil {
		return data, err
	}
	rows, err := a.Pool.Query(ctx, `SELECT id, title, completed FROM issue_subtasks WHERE issue_id=$1 ORDER BY completed, created_at`, data.Item.ID)
	if err != nil {
		return data, err
	}
	defer rows.Close()
	for rows.Next() {
		var item pages.Subtask
		if err = rows.Scan(&item.ID, &item.Title, &item.Completed); err != nil {
			return data, err
		}
		data.Subtasks = append(data.Subtasks, item)
	}
	if err = rows.Err(); err != nil {
		return data, err
	}
	return data, err
}

func (a *App) addSubtask(w http.ResponseWriter, r *http.Request) error {
	item, err := a.findIssue(r)
	if err != nil {
		return err
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" || len([]rune(title)) > 240 {
		return errors.New("a subtarefa deve ter entre 1 e 240 caracteres")
	}
	_, err = a.Pool.Exec(r.Context(), `INSERT INTO issue_subtasks (issue_id,title) VALUES ($1,$2)`, item.ID, title)
	if err != nil {
		return err
	}
	if r.Header.Get("HX-Request") == "true" {
		data, loadErr := a.issueData(r)
		if loadErr != nil {
			return loadErr
		}
		return render(w, r, pages.IssueDetail(data))
	}
	redirect(w, r, issueURL(item.ProjectKey, item.Number))
	return nil
}

func (a *App) toggleSubtask(w http.ResponseWriter, r *http.Request) error {
	item, err := a.findIssue(r)
	if err != nil {
		return err
	}
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		return err
	}
	_, err = a.Pool.Exec(r.Context(), `UPDATE issue_subtasks SET completed=NOT completed, updated_at=now() WHERE id=$1 AND issue_id=$2`, id, item.ID)
	if err != nil {
		return err
	}
	redirect(w, r, issueURL(item.ProjectKey, item.Number))
	return nil
}
func (a *App) issuePage(w http.ResponseWriter, r *http.Request) error {
	data, err := a.issueData(r)
	if err != nil {
		return err
	}
	if r.Header.Get("HX-Request") == "true" {
		return render(w, r, pages.IssueDetail(data))
	}
	return render(w, r, pages.Issue(data))
}
func (a *App) issueResult(w http.ResponseWriter, r *http.Request) error {
	if r.Header.Get("HX-Request") == "true" {
		return a.issuePage(w, r)
	}
	redirect(w, r, "/issues/"+r.PathValue("ref"))
	return nil
}
func (a *App) newIssue(w http.ResponseWriter, r *http.Request) error {
	var data pages.NewIssueData
	var err error
	data.Base, err = a.base(r)
	if err != nil {
		return err
	}
	data.Project, err = a.findProject(r)
	if err != nil {
		return err
	}
	data.Users, err = a.Queries.ListUsers(r.Context(), User(r).WorkspaceID)
	if err != nil {
		return err
	}
	data.Releases, err = a.Queries.ListProjectReleaseOptions(r.Context(), data.Project.ID)
	if err != nil {
		return err
	}
	data.Labels, err = a.Queries.ListLabels(r.Context(), User(r).WorkspaceID)
	if err != nil {
		return err
	}
	return render(w, r, pages.NewIssue(data))
}
func (a *App) createIssue(w http.ResponseWriter, r *http.Request) error {
	project, err := a.findProject(r)
	if err != nil {
		return err
	}
	item, err := a.Issues.Create(r.Context(), User(r), project.ID, r.PostForm)
	if err != nil {
		return err
	}
	redirect(w, r, issueURL(project.Key, item.Number))
	return nil
}
func (a *App) updateIssue(w http.ResponseWriter, r *http.Request) error {
	item, err := a.findIssue(r)
	if err != nil {
		return err
	}
	if err = a.Issues.Update(r.Context(), User(r), item.ID, r.PostForm); err != nil {
		return err
	}
	if r.PostForm.Get("status") == "done" {
		if _, err = a.Pool.Exec(r.Context(), "UPDATE issue_subtasks SET completed=true, updated_at=now() WHERE issue_id=$1", item.ID); err != nil {
			return err
		}
	}
	return a.issueResult(w, r)
}
func (a *App) moveIssue(w http.ResponseWriter, r *http.Request) error {
	item, err := a.findIssue(r)
	if err != nil {
		return err
	}
	for key := range r.PostForm {
		if key != "status" && key != "csrf" {
			return errors.New("apenas o status pode ser alterado por esta ação")
		}
	}
	if r.PostForm.Get("status") == "" {
		return errors.New("o status é obrigatório")
	}
	if err = a.Issues.Update(r.Context(), User(r), item.ID, r.PostForm); err != nil {
		return err
	}
	if r.PostForm.Get("status") == "done" {
		if _, err = a.Pool.Exec(r.Context(), "UPDATE issue_subtasks SET completed=true, updated_at=now() WHERE issue_id=$1", item.ID); err != nil {
			return err
		}
	}
	w.WriteHeader(204)
	return nil
}
func (a *App) createComment(w http.ResponseWriter, r *http.Request) error {
	item, err := a.findIssue(r)
	if err != nil {
		return err
	}
	if err = a.Comments.Create(r.Context(), User(r), item.ID, r.FormValue("body")); err != nil {
		return err
	}
	return a.issueResult(w, r)
}
func (a *App) trackTime(w http.ResponseWriter, r *http.Request) error {
	item, err := a.findIssue(r)
	if err != nil {
		return err
	}
	var minutes int64
	var started time.Time
	if r.FormValue("action") == "manual" {
		minutes, err = strconv.ParseInt(r.FormValue("minutes"), 10, 64)
		if err != nil {
			return errors.New("digite uma quantidade válida de minutos")
		}
		started, err = time.Parse("2006-01-02", r.FormValue("date"))
		if err != nil {
			return errors.New("digite uma data válida")
		}
	}
	if err = a.Time.Track(r.Context(), User(r), item.ID, r.FormValue("action"), r.FormValue("description"), minutes, started); err != nil {
		return err
	}
	redirect(w, r, issueURL(item.ProjectKey, item.Number))
	return nil
}
