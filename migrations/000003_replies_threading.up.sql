ALTER TABLE replies
    ADD COLUMN IF NOT EXISTS parent_reply_id UUID REFERENCES replies (id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS score INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_replies_parent_reply_id ON replies (parent_reply_id);

CREATE TABLE IF NOT EXISTS reply_reactions (
    user_id UUID NOT NULL,
    reply_id UUID NOT NULL REFERENCES replies (id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, reply_id),
    CONSTRAINT reply_reactions_kind_check CHECK (kind IN ('upvote', 'downvote'))
);

CREATE INDEX IF NOT EXISTS idx_reply_reactions_reply_id ON reply_reactions (reply_id);
