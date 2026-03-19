CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    pseudonym TEXT NOT NULL UNIQUE,
    token_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY,
    author_id UUID NOT NULL,
    pseudonym TEXT NOT NULL,
    content TEXT NOT NULL,
    reply_count INTEGER NOT NULL DEFAULT 0,
    score INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT posts_content_len CHECK (char_length(trim(content)) BETWEEN 1 AND 280)
);

CREATE INDEX IF NOT EXISTS idx_posts_author_id ON posts (author_id);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_score_created_at ON posts (score DESC, created_at DESC);

CREATE TABLE IF NOT EXISTS replies (
    id UUID PRIMARY KEY,
    post_id UUID NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
    author_id UUID NOT NULL,
    pseudonym TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT replies_content_len CHECK (char_length(trim(content)) BETWEEN 1 AND 280)
);

CREATE INDEX IF NOT EXISTS idx_replies_post_id_created_at ON replies (post_id, created_at ASC);

CREATE TABLE IF NOT EXISTS reactions (
    user_id UUID NOT NULL,
    post_id UUID NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, post_id),
    CONSTRAINT reactions_kind_check CHECK (kind IN ('upvote', 'downvote'))
);

CREATE INDEX IF NOT EXISTS idx_reactions_post_id ON reactions (post_id);
