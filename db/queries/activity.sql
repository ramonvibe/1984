-- name: CreateActivity :one
INSERT INTO activity_events (workspace_id, project_id, issue_id, actor_id, kind, data)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: ListIssueActivity :many
SELECT a.*, u.name AS actor_name
FROM activity_events a LEFT JOIN users u ON u.id = a.actor_id
WHERE a.issue_id = $1 ORDER BY a.created_at DESC LIMIT 100;

-- name: ListRecentActivity :many
SELECT a.*, u.name AS actor_name, p.key AS project_key, i.number AS issue_number, i.title AS issue_title
FROM activity_events a
LEFT JOIN users u ON u.id = a.actor_id
LEFT JOIN projects p ON p.id = a.project_id
LEFT JOIN issues i ON i.id = a.issue_id
WHERE a.workspace_id = $1 ORDER BY a.created_at DESC LIMIT $2;

