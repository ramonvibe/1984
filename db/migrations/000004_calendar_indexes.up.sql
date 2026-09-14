CREATE INDEX issues_start_date_idx ON issues (start_date) WHERE start_date IS NOT NULL;
CREATE INDEX issues_due_date_idx ON issues (due_date) WHERE due_date IS NOT NULL;
