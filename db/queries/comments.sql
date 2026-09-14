-- name: CreateComment :one
INSERT INTO comments (issue_id, user_id, body) VALUES ($1, $2, $3) RETURNING *;

-- name: ListComments :many
SELECT c.*, u.name AS user_name
FROM comments c JOIN users u ON u.id = c.user_id
WHERE c.issue_id = $1 ORDER BY c.created_at;

