package activity

import (
 "context"
 "encoding/json"
 "strings"
 "github.com/ramon/trackline/internal/database"
)

func Record(ctx context.Context,q *database.Queries,workspace,project,issue,actor int64,kind string,data map[string]string) error {
 if data==nil { data=map[string]string{} }
 encoded,err:=json.Marshal(data); if err!=nil { return err }
 _,err=q.CreateActivity(ctx,database.CreateActivityParams{WorkspaceID:workspace,ProjectID:database.ID(project),IssueID:database.ID(issue),ActorID:database.ID(actor),Kind:kind,Data:encoded})
 return err
}

func Human(kind string,data []byte) string {
 var fields map[string]string
 _=json.Unmarshal(data,&fields)
 names:=map[string]string{"issue.created":"created this issue","issue.updated":"updated this issue","issue.assigned":"changed assignee","issue.status_changed":"changed status","issue.priority_changed":"changed priority","comment.created":"added a comment","timer.started":"started a timer","timer.stopped":"stopped the timer","time.added":"logged time","github.commit_linked":"linked a commit","github.pr_linked":"linked a pull request","github.pr_merged":"merged a pull request","release.created":"created a release","release.published":"published a release","release.updated":"updated a release","release.changelog_saved":"saved the changelog"}
 names["sprint.created"]="created a sprint"
 names["issue.sprint_changed"]="changed sprint"
 result:=names[kind]; if result=="" { result=strings.ReplaceAll(kind,"."," ") }
 if value:=fields["to"]; value!="" { result+=" to "+strings.ReplaceAll(value,"_"," ") }
 if value:=fields["title"]; value!="" { result+=" · "+value }
 return result
}
