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
 names:=map[string]string{"issue.created":"criou esta tarefa","issue.updated":"atualizou esta tarefa","issue.assigned":"alterou a pessoa responsável","issue.status_changed":"alterou o status","issue.priority_changed":"alterou a prioridade","comment.created":"adicionou um comentário","timer.started":"iniciou um cronômetro","timer.stopped":"parou o cronômetro","time.added":"registrou tempo","github.commit_linked":"vinculou um commit","github.pr_linked":"vinculou um pull request","github.pr_merged":"mesclou um pull request","release.created":"criou uma versão","release.published":"publicou uma versão","release.updated":"atualizou uma versão","release.changelog_saved":"salvou o histórico de alterações"}
 names["sprint.created"]="criou uma sprint"
 names["issue.sprint_changed"]="alterou a sprint"
 result:=names[kind]; if result=="" { result=strings.ReplaceAll(kind,"."," ") }
 if value:=fields["to"]; value!="" { result+=" para "+strings.ReplaceAll(value,"_"," ") }
 if value:=fields["title"]; value!="" { result+=" · "+value }
 return result
}
