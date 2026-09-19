ALTER TABLE workspaces
    ADD COLUMN app_name text NOT NULL DEFAULT '1984'
        CHECK (length(trim(app_name)) BETWEEN 1 AND 60),
    ADD COLUMN logo bytea,
    ADD COLUMN logo_content_type text,
    ADD CONSTRAINT workspaces_logo_size_check
        CHECK (logo IS NULL OR octet_length(logo) <= 1048576),
    ADD CONSTRAINT workspaces_logo_type_check
        CHECK (logo_content_type IS NULL OR logo_content_type IN ('image/png', 'image/jpeg', 'image/webp', 'image/gif')),
    ADD CONSTRAINT workspaces_logo_pair_check
        CHECK ((logo IS NULL) = (logo_content_type IS NULL));
