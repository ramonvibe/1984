-- name: ReportGroups :many
SELECT
 CASE sqlc.arg(group_by)::text
 WHEN 'member' THEN u.name || ' (' || u.email || ')' WHEN 'project' THEN p.key || ' · ' || p.name WHEN 'release' THEN coalesce(p.key || ' · ' || r.version, 'No release')
 WHEN 'type' THEN i.type ELSE p.key || '-' || i.number::text || ' · ' || i.title END::text AS label,
 sum(CASE WHEN t.ended_at IS NULL THEN extract(epoch FROM (now()-t.started_at))::bigint ELSE t.duration_seconds::bigint END)::bigint AS seconds
FROM time_entries t JOIN issues i ON i.id=t.issue_id JOIN projects p ON p.id=i.project_id
JOIN users u ON u.id=t.user_id LEFT JOIN releases r ON r.id=i.release_id
WHERE p.workspace_id=sqlc.arg(workspace_id)
 AND (sqlc.arg(project_id)::bigint=0 OR p.id=sqlc.arg(project_id))
 AND (sqlc.narg(date_from)::date IS NULL OR t.started_at >= sqlc.narg(date_from)::date)
 AND (sqlc.narg(date_to)::date IS NULL OR t.started_at < sqlc.narg(date_to)::date + interval '1 day')
GROUP BY 1 ORDER BY seconds DESC LIMIT 200;

-- name: SearchProjectsAndReleases :many
SELECT p.name AS title, '/projects/' || p.key AS url, 'Project'::text AS kind
FROM projects p WHERE p.workspace_id=$1 AND (p.name ILIKE '%' || $2 || '%' OR p.key ILIKE '%' || $2 || '%')
UNION ALL
SELECT r.version || ' · ' || r.name, '/releases/' || r.id::text, 'Release'::text
FROM releases r JOIN projects p ON p.id=r.project_id WHERE p.workspace_id=$1 AND (r.version ILIKE '%' || $2 || '%' OR r.name ILIKE '%' || $2 || '%')
LIMIT 30;
