CREATE TABLE workspace_invites (
    token_hash bytea PRIMARY KEY,
    workspace_id bigint NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_by bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX workspace_invites_expiry_idx ON workspace_invites(expires_at);
