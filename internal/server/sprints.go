package server

import (
	"errors"
	"net/http"
	"strconv"
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
	redirect(w, r, "/projects/"+project.Key+"/board?sprint="+strconv.FormatInt(sprint.ID, 10))
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
			return errors.New("invalid sprint")
		}
	}
	if err = a.Projects.SetIssueSprint(r.Context(), User(r), item.ID, id); err != nil {
		return err
	}
	return a.issueResult(w, r)
}
