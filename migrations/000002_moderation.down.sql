DROP TABLE IF EXISTS reports;

DROP INDEX IF EXISTS idx_posts_is_hidden_created_at;

ALTER TABLE posts
    DROP COLUMN IF EXISTS is_hidden;

ALTER TABLE users
    DROP COLUMN IF EXISTS is_admin;
