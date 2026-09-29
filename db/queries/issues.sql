-- name: CreateIssue :one
INSERT INTO issues (project_id, number, title, description, type, status, priority, assignee_id, reporter_id, estimated_minutes, start_date, due_date, release_id, closed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, CASE WHEN $6 IN ('done','canceled') THEN now() ELSE NULL END)
RETURNING *;

-- name: GetIssueByKey :one
SELECT i.*, p.key AS project_key, p.name AS project_name,
       assignee.name AS assignee_name, reporter.name AS reporter_name,
       r.version AS release_version
FROM issues i
JOIN projects p ON p.id = i.project_id
JOIN users reporter ON reporter.id = i.reporter_id
LEFT JOIN users assignee ON assignee.id = i.assignee_id
LEFT JOIN releases r ON r.id = i.release_id
WHERE p.workspace_id = $1 AND p.key = upper(sqlc.arg(key)::text) AND i.number = $2;

-- name: GetIssue :one
SELECT * FROM issues WHERE id = $1;

-- name: LockIssue :one
SELECT i.* FROM issues i JOIN projects p ON p.id = i.project_id
WHERE i.id = $1 AND p.workspace_id = $2 FOR UPDATE OF i;

-- name: UpdateIssue :one
UPDATE issues SET title = $2, description = $3, type = $4, status = $5,
    priority = $6, assignee_id = $7, estimated_minutes = $8, start_date = $9,
    due_date = $10, release_id = $11, updated_at = now(),
    closed_at = CASE WHEN $5 IN ('done', 'canceled') THEN coalesce(closed_at, now()) ELSE NULL END
WHERE id = $1 RETURNING *;

-- name: ListProjectIssues :many
SELECT i.*, p.key AS project_key, assignee.name AS assignee_name,
       coalesce(string_agg(DISTINCT l.name, ',' ORDER BY l.name), '') AS label_names
FROM issues i
JOIN projects p ON p.id = i.project_id
LEFT JOIN users assignee ON assignee.id = i.assignee_id
LEFT JOIN issue_labels il ON il.issue_id = i.id
LEFT JOIN labels l ON l.id = il.label_id
WHERE i.project_id = $1
  AND ($2::text = '' OR i.status = $2)
  AND ($3::text = '' OR i.type = $3)
GROUP BY i.id, p.key, assignee.name
ORDER BY CASE i.priority WHEN 'urgent' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END, i.created_at DESC
LIMIT $4 OFFSET $5;

-- name: ListBoardIssues :many
SELECT i.id, i.number, i.title, i.type, i.status, i.priority, p.key AS project_key,
       assignee.id AS assignee_id, assignee.name AS assignee_name,
       coalesce(string_agg(DISTINCT l.name, ',' ORDER BY l.name), '') AS label_names
FROM issues i
JOIN projects p ON p.id = i.project_id
LEFT JOIN users assignee ON assignee.id = i.assignee_id
LEFT JOIN issue_labels il ON il.issue_id = i.id
LEFT JOIN labels l ON l.id = il.label_id
WHERE i.project_id = $1 AND i.status <> 'canceled'
  AND (sqlc.arg(sprint_filter)::bigint = 0
       OR (sqlc.arg(sprint_filter)::bigint = -1 AND i.sprint_id IS NULL)
       OR i.sprint_id = sqlc.arg(sprint_filter)::bigint)
GROUP BY i.id, p.key, assignee.id, assignee.name
ORDER BY i.updated_at DESC LIMIT 500;

-- name: SetIssueLabels :exec
DELETE FROM issue_labels WHERE issue_id = $1;

-- name: AddIssueLabel :exec
INSERT INTO issue_labels (issue_id, label_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListIssueLabels :many
SELECT l.* FROM labels l JOIN issue_labels il ON il.label_id = l.id
WHERE il.issue_id = $1 ORDER BY l.name;

-- name: SearchIssues :many
SELECT i.id, i.number, i.title, i.type, i.status, i.priority, p.key AS project_key, p.name AS project_name
FROM issues i JOIN projects p ON p.id = i.project_id
LEFT JOIN releases r ON r.id = i.release_id
WHERE p.workspace_id = $1 AND (
    lower(i.title) LIKE '%' || lower($2) || '%' OR
    lower(p.name) LIKE '%' || lower($2) || '%' OR
    lower(p.key || '-' || i.number::text) = lower($2) OR
    lower(coalesce(r.version, '')) LIKE '%' || lower($2) || '%'
)
ORDER BY i.updated_at DESC LIMIT 30;

-- name: ListCalendarItems :many
SELECT i.id, i.number, i.title, p.key AS project_key, i.start_date, i.due_date,
       NULL::date AS target_date, 'issue'::text AS item_kind
FROM issues i JOIN projects p ON p.id = i.project_id
WHERE p.workspace_id = $1 AND (i.start_date IS NOT NULL OR i.due_date IS NOT NULL)
UNION ALL
SELECT r.id, 0, coalesce(nullif(r.name, ''), r.version), p.key, NULL::date, NULL::date,
       r.target_date, 'release'::text
FROM releases r JOIN projects p ON p.id = r.project_id
WHERE p.workspace_id = $1 AND r.target_date IS NOT NULL
ORDER BY target_date NULLS LAST, due_date NULLS LAST, start_date NULLS LAST;
