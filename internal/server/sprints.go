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
