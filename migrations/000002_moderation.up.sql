ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS is_hidden BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_posts_is_hidden_created_at ON posts (is_hidden, created_at DESC);

CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY,
    post_id UUID NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
    reporter_id UUID NOT NULL,
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    action TEXT,
    action_note TEXT NOT NULL DEFAULT '',
    reviewed_by UUID,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT reports_status_check CHECK (status IN ('open', 'resolved')),
    CONSTRAINT reports_action_check CHECK (action IS NULL OR action IN ('dismiss', 'hide', 'ban')),
    CONSTRAINT reports_reason_len CHECK (char_length(trim(reason)) BETWEEN 1 AND 500),
    CONSTRAINT reports_unique_reporter_per_post UNIQUE (post_id, reporter_id)
);

CREATE INDEX IF NOT EXISTS idx_reports_status_created_at ON reports (status, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_reports_post_id ON reports (post_id);
