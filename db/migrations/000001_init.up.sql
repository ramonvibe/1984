CREATE TABLE workspaces (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 120),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id bigint NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 120),
    email text NOT NULL,
    password_hash text NOT NULL,
    role text NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_workspace_email_uidx ON users (workspace_id, lower(email));

CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE projects (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id bigint NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 120),
    key text NOT NULL CHECK (key ~ '^[A-Z][A-Z0-9]{1,9}$'),
    description text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    next_issue_number integer NOT NULL DEFAULT 1 CHECK (next_issue_number > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, key)
);
CREATE INDEX projects_workspace_id_idx ON projects (workspace_id);

CREATE TABLE project_members (
    project_id bigint NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);

CREATE TABLE releases (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id bigint NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    version text NOT NULL CHECK (length(trim(version)) BETWEEN 1 AND 80),
    name text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    changelog text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'planning' CHECK (status IN ('planning', 'active', 'released', 'canceled')),
    target_date date,
    published_at timestamptz,
    github_release_id bigint,
    github_tag text,
    github_url text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, version)
);
CREATE INDEX releases_project_id_idx ON releases (project_id);
CREATE INDEX releases_target_date_idx ON releases (target_date);

CREATE TABLE issues (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id bigint NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number integer NOT NULL CHECK (number > 0),
    title text NOT NULL CHECK (length(trim(title)) BETWEEN 1 AND 240),
    description text NOT NULL DEFAULT '',
    type text NOT NULL DEFAULT 'task' CHECK (type IN ('bug', 'feature', 'improvement', 'task')),
    status text NOT NULL DEFAULT 'backlog' CHECK (status IN ('backlog', 'todo', 'in_progress', 'review', 'done', 'canceled')),
    priority text NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    assignee_id bigint REFERENCES users(id) ON DELETE SET NULL,
    reporter_id bigint NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    estimated_minutes integer NOT NULL DEFAULT 0 CHECK (estimated_minutes >= 0),
    start_date date,
    due_date date,
    release_id bigint REFERENCES releases(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    closed_at timestamptz,
    UNIQUE (project_id, number),
    CHECK (due_date IS NULL OR start_date IS NULL OR due_date >= start_date)
);
CREATE INDEX issues_project_id_idx ON issues (project_id);
CREATE INDEX issues_status_idx ON issues (status);
CREATE INDEX issues_assignee_id_idx ON issues (assignee_id);
CREATE INDEX issues_release_id_idx ON issues (release_id);
CREATE INDEX issues_created_at_idx ON issues (created_at DESC);
CREATE INDEX issues_project_status_idx ON issues (project_id, status);

CREATE TABLE labels (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id bigint NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 40),
    color text NOT NULL DEFAULT '#64748b' CHECK (color ~ '^#[0-9a-fA-F]{6}$'),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX labels_workspace_name_uidx ON labels (workspace_id, lower(name));

CREATE TABLE issue_labels (
    issue_id bigint NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    label_id bigint NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, label_id)
);

CREATE TABLE comments (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    issue_id bigint NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body text NOT NULL CHECK (length(trim(body)) BETWEEN 1 AND 10000),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX comments_issue_id_idx ON comments (issue_id, created_at);

CREATE TABLE time_entries (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    issue_id bigint NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    started_at timestamptz NOT NULL,
    ended_at timestamptz,
    duration_seconds integer CHECK (duration_seconds IS NULL OR duration_seconds >= 0),
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((ended_at IS NULL AND duration_seconds IS NULL) OR (ended_at IS NOT NULL AND duration_seconds IS NOT NULL))
);
CREATE UNIQUE INDEX time_entries_one_active_per_user_uidx ON time_entries (user_id) WHERE ended_at IS NULL;
CREATE INDEX time_entries_issue_id_idx ON time_entries (issue_id);
CREATE INDEX time_entries_user_id_idx ON time_entries (user_id);
CREATE INDEX time_entries_started_at_idx ON time_entries (started_at DESC);

CREATE TABLE activity_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id bigint NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id bigint REFERENCES projects(id) ON DELETE CASCADE,
    issue_id bigint REFERENCES issues(id) ON DELETE CASCADE,
    actor_id bigint REFERENCES users(id) ON DELETE SET NULL,
    kind text NOT NULL,
    data jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX activity_events_issue_idx ON activity_events (issue_id, created_at DESC);
CREATE INDEX activity_events_project_idx ON activity_events (project_id, created_at DESC);
CREATE INDEX activity_events_workspace_idx ON activity_events (workspace_id, created_at DESC);

CREATE TABLE github_installations (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id bigint NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    installation_id bigint NOT NULL UNIQUE,
    account_login text NOT NULL DEFAULT '',
    account_id bigint,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE github_repositories (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id bigint NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE,
    installation_id bigint NOT NULL REFERENCES github_installations(id) ON DELETE CASCADE,
    repository_id bigint NOT NULL UNIQUE,
    owner text NOT NULL,
    name text NOT NULL,
    full_name text NOT NULL,
    default_branch text NOT NULL DEFAULT 'main',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE github_commits (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    repository_id bigint NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    issue_id bigint NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    sha text NOT NULL,
    message text NOT NULL,
    url text NOT NULL DEFAULT '',
    author_name text NOT NULL DEFAULT '',
    authored_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (repository_id, sha, issue_id)
);
CREATE INDEX github_commits_issue_id_idx ON github_commits (issue_id, authored_at DESC);

CREATE TABLE github_pull_requests (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    repository_id bigint NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    issue_id bigint NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    number integer NOT NULL,
    title text NOT NULL,
    state text NOT NULL,
    merged boolean NOT NULL DEFAULT false,
    url text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (repository_id, number, issue_id)
);
CREATE INDEX github_pull_requests_issue_id_idx ON github_pull_requests (issue_id, updated_at DESC);

