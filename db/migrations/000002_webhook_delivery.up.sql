CREATE TABLE github_deliveries (
 delivery_id text PRIMARY KEY,
 event text NOT NULL,
 processed_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX github_deliveries_processed_idx ON github_deliveries(processed_at);
ALTER TABLE time_entries ADD CONSTRAINT time_entries_chronological CHECK (ended_at IS NULL OR ended_at >= started_at);
