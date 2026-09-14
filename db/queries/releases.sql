-- name: CreateRelease :one
INSERT INTO releases (project_id, version, name, description, status, target_date)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: ListReleases :many
SELECT r.*,
       count(i.id)::bigint AS issue_count,
       count(i.id) FILTER (WHERE i.status = 'done')::bigint AS done_count,
       coalesce(sum(i.estimated_minutes), 0)::bigint AS estimated_minutes,
       coalesce(sum(te.seconds), 0)::bigint AS spent_seconds
FROM releases r
LEFT JOIN issues i ON i.release_id = r.id
LEFT JOIN LATERAL (
    SELECT sum(CASE WHEN ended_at IS NULL THEN extract(epoch FROM (now() - started_at))::bigint ELSE duration_seconds::bigint END)::bigint AS seconds
    FROM time_entries WHERE issue_id = i.id
) te ON true
WHERE r.project_id = $1 GROUP BY r.id ORDER BY r.target_date NULLS LAST, r.created_at DESC;

-- name: GetRelease :one
SELECT r.*, p.key AS project_key, p.name AS project_name,
       count(i.id)::bigint AS issue_count,
       count(i.id) FILTER (WHERE i.status = 'done')::bigint AS done_count,
       count(i.id) FILTER (WHERE i.status NOT IN ('done', 'canceled'))::bigint AS open_count,
       count(i.id) FILTER (WHERE i.type = 'bug')::bigint AS bug_count,
       count(i.id) FILTER (WHERE i.type = 'feature')::bigint AS feature_count,
       count(i.id) FILTER (WHERE i.type = 'task')::bigint AS task_count,
       coalesce(sum(i.estimated_minutes), 0)::bigint AS estimated_minutes
FROM releases r JOIN projects p ON p.id = r.project_id LEFT JOIN issues i ON i.release_id = r.id
WHERE r.id = $1 AND p.workspace_id = $2 GROUP BY r.id, p.key, p.name;

-- name: ListReleaseIssues :many
SELECT i.*, p.key AS project_key FROM issues i JOIN projects p ON p.id = i.project_id
WHERE i.release_id = $1 ORDER BY i.type, i.number;

-- name: SaveReleaseChangelog :one
UPDATE releases SET changelog = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: MarkReleasePublished :one
UPDATE releases SET status = 'released', github_release_id = $2, github_url = $3,
    github_tag = $4, published_at = now(), updated_at = now()
WHERE id = $1 RETURNING *;

-- name: ListProjectReleaseOptions :many
SELECT id, version FROM releases WHERE project_id = $1 AND status <> 'canceled' ORDER BY created_at DESC;

-- name: UpdateRelease :one
UPDATE releases SET version = $2, name = $3, description = $4, status = $5, target_date = $6, updated_at = now()
WHERE id = $1 AND github_release_id IS NULL RETURNING *;

-- name: LockRelease :one
SELECT * FROM releases WHERE id = $1 FOR UPDATE;

-- name: ReleaseSpent :one
SELECT coalesce(sum(CASE WHEN t.ended_at IS NULL THEN extract(epoch FROM (now()-t.started_at))::bigint ELSE t.duration_seconds::bigint END),0)::bigint
FROM time_entries t JOIN issues i ON i.id = t.issue_id WHERE i.release_id = $1;
