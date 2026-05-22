DROP INDEX IF EXISTS idx_posts_search_vector;
ALTER TABLE posts DROP COLUMN IF EXISTS search_vector;
DROP INDEX IF EXISTS idx_outbox_events_unprocessed;
DROP TABLE IF EXISTS outbox_events;
