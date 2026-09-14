-- name: DashboardStats :one
SELECT
  count(i.id) FILTER (WHERE i.status NOT IN ('done', 'canceled'))::bigint AS open_issues,
  count(i.id) FILTER (WHERE i.status = 'done')::bigint AS completed_issues,
  count(i.id) FILTER (WHERE i.type = 'bug' AND i.status NOT IN ('done', 'canceled'))::bigint AS open_bugs,
  coalesce((SELECT sum(CASE WHEN te.ended_at IS NULL THEN extract(epoch FROM (now() - te.started_at))::bigint ELSE te.duration_seconds::bigint END)
            FROM time_entries te JOIN users u ON u.id = te.user_id WHERE u.workspace_id = $1), 0)::bigint AS tracked_seconds
FROM issues i JOIN projects p ON p.id = i.project_id WHERE p.workspace_id = $1;

-- name: RecentIssues :many
SELECT i.id, i.number, i.title, i.status, i.type, i.priority, i.updated_at, p.key AS project_key
FROM issues i JOIN projects p ON p.id = i.project_id
WHERE p.workspace_id = $1 ORDER BY i.updated_at DESC LIMIT $2;

-- name: CurrentRelease :one
SELECT r.id, r.version, r.name, r.target_date, p.key AS project_key,
       count(i.id)::bigint AS issue_count,
       count(i.id) FILTER (WHERE i.status = 'done')::bigint AS done_count
FROM releases r JOIN projects p ON p.id = r.project_id LEFT JOIN issues i ON i.release_id = r.id
WHERE p.workspace_id = $1 AND r.status = 'active'
GROUP BY r.id, p.key ORDER BY r.target_date NULLS LAST LIMIT 1;

