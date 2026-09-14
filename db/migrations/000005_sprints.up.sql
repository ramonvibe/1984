CREATE TABLE sprints (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id bigint NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(name) BETWEEN 1 AND 120),
    start_date date NOT NULL,
    end_date date NOT NULL CHECK (end_date >= start_date),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, project_id)
);
CREATE INDEX sprints_project_id_idx ON sprints(project_id);
ALTER TABLE issues ADD COLUMN sprint_id bigint;
ALTER TABLE issues ADD CONSTRAINT issues_sprint_project_fk
    FOREIGN KEY (sprint_id, project_id) REFERENCES sprints(id, project_id);
CREATE INDEX issues_sprint_id_idx ON issues(sprint_id) WHERE sprint_id IS NOT NULL;
