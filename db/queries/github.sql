-- name: UpsertGitHubInstallation :one
INSERT INTO github_installations (workspace_id, installation_id, account_login, account_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (installation_id) DO UPDATE SET account_login = excluded.account_login, account_id = excluded.account_id, updated_at = now()
RETURNING *;

-- name: LinkGitHubRepository :one
INSERT INTO github_repositories (project_id, installation_id, repository_id, owner, name, full_name, default_branch)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (project_id) DO UPDATE SET installation_id = excluded.installation_id, repository_id = excluded.repository_id,
owner = excluded.owner, name = excluded.name, full_name = excluded.full_name, default_branch = excluded.default_branch, updated_at = now()
RETURNING *;

-- name: GetGitHubRepositoryByProject :one
SELECT gr.*, gi.installation_id AS github_installation_id
FROM github_repositories gr JOIN github_installations gi ON gi.id = gr.installation_id
WHERE gr.project_id = $1;

-- name: GetGitHubRepositoryByExternalID :one
SELECT gr.*, p.workspace_id, p.key AS project_key
FROM github_repositories gr JOIN projects p ON p.id = gr.project_id
WHERE gr.repository_id = $1;

-- name: FindIssueReference :one
SELECT i.* FROM issues i JOIN projects p ON p.id = i.project_id
WHERE p.workspace_id = $1 AND p.key = upper($2) AND i.number = $3;

-- name: UpsertGitHubCommit :execrows
INSERT INTO github_commits (repository_id, issue_id, sha, message, url, author_name, authored_at)
VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING;

-- name: UpsertGitHubPullRequest :execrows
INSERT INTO github_pull_requests (repository_id, issue_id, number, title, state, merged, url)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (repository_id, number, issue_id) DO UPDATE SET title = excluded.title, state = excluded.state,
merged = excluded.merged, url = excluded.url, updated_at = now()
WHERE (github_pull_requests.title,github_pull_requests.state,github_pull_requests.merged,github_pull_requests.url)
IS DISTINCT FROM (excluded.title,excluded.state,excluded.merged,excluded.url);

-- name: ListIssueCommits :many
SELECT * FROM github_commits WHERE issue_id = $1 ORDER BY authored_at DESC LIMIT 50;

-- name: ListIssuePullRequests :many
SELECT * FROM github_pull_requests WHERE issue_id = $1 ORDER BY updated_at DESC LIMIT 50;

-- name: ReleaseGitHubCounts :one
SELECT
 (SELECT count(DISTINCT pr.number)::bigint FROM github_pull_requests pr JOIN issues i ON i.id=pr.issue_id WHERE i.release_id=$1) AS pull_requests,
 (SELECT count(DISTINCT c.sha)::bigint FROM github_commits c JOIN issues i ON i.id=c.issue_id WHERE i.release_id=$1) AS commits;

-- name: ClaimGitHubDelivery :execrows
INSERT INTO github_deliveries (delivery_id,event) VALUES ($1,$2) ON CONFLICT DO NOTHING;

-- name: DeleteGitHubInstallation :exec
DELETE FROM github_installations WHERE installation_id=$1;

-- name: DeleteOldGitHubDeliveries :exec
DELETE FROM github_deliveries WHERE processed_at < now() - interval '30 days';

-- name: UpdateGitHubReleaseFromWebhook :exec
UPDATE releases SET github_url=$2, status=$3, updated_at=now()
WHERE github_release_id=$1 AND project_id=$4;
