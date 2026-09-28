-- name: WorkspaceCount :one
SELECT count(*) FROM workspaces;

-- name: CreateWorkspace :one
INSERT INTO workspaces (name) VALUES ($1) RETURNING *;

-- name: GetWorkspace :one
SELECT * FROM workspaces WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (workspace_id, name, email, password_hash, role)
VALUES ($1, $2, sqlc.arg(email)::text, $3, $4) RETURNING *;

-- name: ListUsers :many
SELECT id, workspace_id, name, email, role, active, created_at, updated_at
FROM users WHERE workspace_id = $1 ORDER BY name;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE lower(email) = lower($1) AND active = true LIMIT 1;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3);

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < now();

-- name: GetSessionUser :one
SELECT u.id, u.workspace_id, u.name, u.email, u.role, u.active, u.created_at, u.updated_at,
       w.name AS workspace_name, w.app_name, w.logo IS NOT NULL AS has_logo, w.sprints_are_releases, u.avatar IS NOT NULL AS has_avatar
FROM sessions s
JOIN users u ON u.id = s.user_id
JOIN workspaces w ON w.id = u.workspace_id
WHERE s.token_hash = $1 AND s.expires_at > now() AND u.active = true;

-- name: CreateLabel :one
INSERT INTO labels (workspace_id, name, color) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING RETURNING *;

-- name: ListLabels :many
SELECT * FROM labels WHERE workspace_id = $1 ORDER BY name;

-- name: GetWorkspaceUser :one
SELECT * FROM users WHERE id = $1 AND workspace_id = $2 AND active = true;

-- name: GetWorkspaceLabel :one
SELECT * FROM labels WHERE id = $1 AND workspace_id = $2;

-- name: LockOnboarding :exec
SELECT pg_advisory_xact_lock(1984001);
