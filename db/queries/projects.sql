-- name: CreateProject :one
INSERT INTO projects (workspace_id, name, key, description)
VALUES ($1, $2, sqlc.arg(key)::text, $3) RETURNING *;

-- name: AddProjectMember :exec
INSERT INTO project_members (project_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListProjectMembers :many
SELECT u.id, u.name, u.email, u.role
FROM users u JOIN project_members pm ON pm.user_id = u.id
WHERE pm.project_id = $1 ORDER BY u.name;

-- name: ListProjects :many
SELECT p.*,
       count(i.id) FILTER (WHERE i.status NOT IN ('done', 'canceled'))::bigint AS open_issues,
       count(i.id) FILTER (WHERE i.status = 'done')::bigint AS done_issues
FROM projects p LEFT JOIN issues i ON i.project_id = p.id
WHERE p.workspace_id = $1
GROUP BY p.id ORDER BY p.name;

-- name: GetProjectByKey :one
SELECT * FROM projects WHERE workspace_id = $1 AND key = upper(sqlc.arg(key)::text);

-- name: GetProject :one
SELECT * FROM projects WHERE id = $1 AND workspace_id = $2;

-- name: UpdateProject :one
UPDATE projects SET name = $2, description = $3, status = $4, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: ReserveIssueNumber :one
UPDATE projects SET next_issue_number = next_issue_number + 1, updated_at = now()
WHERE id = $1 RETURNING next_issue_number - 1;

-- name: RemoveProjectMember :exec
DELETE FROM project_members WHERE project_id = $1 AND user_id = $2;
