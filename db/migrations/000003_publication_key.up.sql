ALTER TABLE releases ADD COLUMN publication_key text NOT NULL DEFAULT gen_random_uuid()::text;
