-- name: StartTimer :one
INSERT INTO time_entries (issue_id, user_id, started_at, description)
VALUES ($1, $2, now(), $3) RETURNING *;

-- name: StopTimer :one
UPDATE time_entries SET ended_at = now(), duration_seconds = greatest(0, extract(epoch FROM (now() - started_at))::integer)
WHERE user_id = $1 AND issue_id = $2 AND ended_at IS NULL RETURNING *;

-- name: AddTimeEntry :one
INSERT INTO time_entries (issue_id, user_id, started_at, ended_at, duration_seconds, description)
VALUES (sqlc.arg(issue_id), sqlc.arg(user_id), sqlc.arg(started_at)::timestamptz,
        sqlc.arg(started_at)::timestamptz + make_interval(secs => sqlc.arg(duration_seconds)::double precision),
        sqlc.arg(duration_seconds), sqlc.arg(description)) RETURNING *;

-- name: GetActiveTimer :one
SELECT te.*, i.number AS issue_number, i.title AS issue_title, p.key AS project_key
FROM time_entries te JOIN issues i ON i.id = te.issue_id JOIN projects p ON p.id = i.project_id
WHERE te.user_id = $1 AND te.ended_at IS NULL;

-- name: ListIssueTime :many
SELECT te.*, u.name AS user_name
FROM time_entries te JOIN users u ON u.id = te.user_id
WHERE te.issue_id = $1 ORDER BY te.started_at DESC LIMIT 100;

-- name: IssueTimeSummary :many
SELECT u.id AS user_id, u.name, coalesce(sum(
    CASE WHEN te.ended_at IS NULL THEN extract(epoch FROM (now() - te.started_at))::bigint
         ELSE te.duration_seconds::bigint END), 0)::bigint AS seconds
FROM time_entries te JOIN users u ON u.id = te.user_id
WHERE te.issue_id = $1 GROUP BY u.id, u.name ORDER BY seconds DESC;

-- name: ProjectTimeReport :many
SELECT i.id AS issue_id, p.key AS project_key, i.number, i.title, i.type, i.estimated_minutes,
       coalesce(sum(CASE WHEN te.ended_at IS NULL THEN extract(epoch FROM (now() - te.started_at))::bigint ELSE te.duration_seconds::bigint END), 0)::bigint AS spent_seconds
FROM issues i JOIN projects p ON p.id = i.project_id
LEFT JOIN time_entries te ON te.issue_id = i.id
 AND ($2::date IS NULL OR te.started_at >= $2::date) AND ($3::date IS NULL OR te.started_at < $3::date + interval '1 day')
WHERE i.project_id = $1
GROUP BY i.id, p.key ORDER BY spent_seconds DESC, i.number DESC LIMIT 500;

-- name: TimeByMember :many
SELECT u.name AS label, coalesce(sum(CASE WHEN te.ended_at IS NULL THEN extract(epoch FROM (now() - te.started_at))::bigint ELSE te.duration_seconds::bigint END), 0)::bigint AS seconds
FROM users u LEFT JOIN time_entries te ON te.user_id = u.id
WHERE u.workspace_id = $1 GROUP BY u.id ORDER BY seconds DESC;

-- name: TimeByProject :many
SELECT p.name AS label, coalesce(sum(te.duration_seconds), 0)::bigint AS seconds
FROM projects p LEFT JOIN issues i ON i.project_id = p.id LEFT JOIN time_entries te ON te.issue_id = i.id
WHERE p.workspace_id = $1 GROUP BY p.id ORDER BY seconds DESC;

-- name: TimeByIssueType :many
SELECT i.type AS label, coalesce(sum(te.duration_seconds), 0)::bigint AS seconds
FROM issues i JOIN projects p ON p.id = i.project_id LEFT JOIN time_entries te ON te.issue_id = i.id
WHERE p.workspace_id = $1 GROUP BY i.type ORDER BY seconds DESC;

-- name: TimeByRelease :many
SELECT r.version AS label, coalesce(sum(te.duration_seconds), 0)::bigint AS seconds
FROM releases r JOIN projects p ON p.id = r.project_id LEFT JOIN issues i ON i.release_id = r.id LEFT JOIN time_entries te ON te.issue_id = i.id
WHERE p.workspace_id = $1 GROUP BY r.id ORDER BY seconds DESC;
