ALTER TABLE users DROP CONSTRAINT users_avatar_pair_check, DROP CONSTRAINT users_avatar_type_check, DROP CONSTRAINT users_avatar_size_check, DROP COLUMN avatar_content_type, DROP COLUMN avatar;
