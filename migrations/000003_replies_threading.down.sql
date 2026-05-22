DROP INDEX IF EXISTS idx_reply_reactions_reply_id;
DROP TABLE IF EXISTS reply_reactions;

DROP INDEX IF EXISTS idx_replies_parent_reply_id;

ALTER TABLE replies
    DROP COLUMN IF EXISTS score,
    DROP COLUMN IF EXISTS parent_reply_id;
