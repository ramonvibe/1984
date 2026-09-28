ALTER TABLE workspaces DROP CONSTRAINT workspaces_logo_size_check;
ALTER TABLE workspaces ADD CONSTRAINT workspaces_logo_size_check
    CHECK (logo IS NULL OR octet_length(logo) <= 5242880);
