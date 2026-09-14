-- name: CreateSprint :one
INSERT INTO sprints (project_id, name, start_date, end_date)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: ListProjectSprints :many
SELECT * FROM sprints WHERE project_id = $1 ORDER BY start_date DESC, id DESC;

-- name: GetProjectSprint :one
SELECT * FROM sprints WHERE id = $1 AND project_id = $2;

-- name: SetIssueSprint :exec
UPDATE issues SET sprint_id = $2, updated_at = now() WHERE id = $1;
