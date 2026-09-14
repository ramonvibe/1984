package server

import (
 "net/http"
 "github.com/ramon/trackline/internal/database"
 "github.com/ramon/trackline/web/pages"
)

func (a *App) releasePage(w http.ResponseWriter,r *http.Request) error {
 id,err:=parseID(r.PathValue("id")); if err!=nil { return err }
 var data pages.ReleaseData
 data.Base,err=a.base(r); if err!=nil { return err }
 data.Item,err=a.Queries.GetRelease(r.Context(),database.GetReleaseParams{ID:id,WorkspaceID:User(r).WorkspaceID}); if err!=nil { return err }
 data.Issues,err=a.Queries.ListReleaseIssues(r.Context(),database.ID(id)); if err!=nil { return err }
 data.Counts,err=a.Queries.ReleaseGitHubCounts(r.Context(),database.ID(id)); if err!=nil { return err }
 data.Spent,err=a.Queries.ReleaseSpent(r.Context(),database.ID(id)); if err!=nil { return err }
 data.GitHubEnabled=a.GitHub.Client.Enabled()
 return render(w,r,pages.Release(data))
}
func (a *App) createRelease(w http.ResponseWriter,r *http.Request) error {
 project,err:=a.findProject(r); if err!=nil { return err }
 item,err:=a.Releases.Create(r.Context(),User(r),project.ID,r.PostForm); if err!=nil { return err }
 redirect(w,r,releaseURL(item.ID)); return nil
}
func (a *App) updateRelease(w http.ResponseWriter,r *http.Request) error {
 id,err:=parseID(r.PathValue("id")); if err!=nil { return err }
 if err=a.Releases.Update(r.Context(),User(r),id,r.PostForm); err!=nil { return err }
 redirect(w,r,releaseURL(id)); return nil
}
func (a *App) generateChangelog(w http.ResponseWriter,r *http.Request) error { return a.changelog(w,r,true) }
func (a *App) saveChangelog(w http.ResponseWriter,r *http.Request) error { return a.changelog(w,r,false) }
func (a *App) changelog(w http.ResponseWriter,r *http.Request,generate bool) error {
 id,err:=parseID(r.PathValue("id")); if err!=nil { return err }
 if err=a.Releases.Save(r.Context(),User(r),id,r.FormValue("changelog"),generate); err!=nil { return err }
 redirect(w,r,releaseURL(id)); return nil
}
func (a *App) publishRelease(w http.ResponseWriter,r *http.Request) error {
 id,err:=parseID(r.PathValue("id")); if err!=nil { return err }
 if err=a.Releases.Publish(r.Context(),User(r),id); err!=nil { return err }
 redirect(w,r,releaseURL(id)); return nil
}
