ALTER TABLE users
    ADD COLUMN avatar bytea,
    ADD COLUMN avatar_content_type text,
    ADD CONSTRAINT users_avatar_size_check CHECK (avatar IS NULL OR octet_length(avatar) <= 5242880),
    ADD CONSTRAINT users_avatar_type_check CHECK (avatar_content_type IS NULL OR avatar_content_type IN ('image/png','image/jpeg','image/webp','image/gif')),
    ADD CONSTRAINT users_avatar_pair_check CHECK ((avatar IS NULL) = (avatar_content_type IS NULL));
