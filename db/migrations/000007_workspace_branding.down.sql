ALTER TABLE workspaces
    DROP CONSTRAINT workspaces_logo_pair_check,
    DROP CONSTRAINT workspaces_logo_type_check,
    DROP CONSTRAINT workspaces_logo_size_check,
    DROP COLUMN logo_content_type,
    DROP COLUMN logo,
    DROP COLUMN app_name;
