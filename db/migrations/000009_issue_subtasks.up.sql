CREATE TABLE issue_subtasks (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    issue_id bigint NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    title text NOT NULL CHECK (length(trim(title)) BETWEEN 1 AND 240),
    completed boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX issue_subtasks_issue_idx ON issue_subtasks (issue_id, completed, created_at);
