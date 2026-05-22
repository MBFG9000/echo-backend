CREATE TABLE IF NOT EXISTS post_attachments (
    id UUID PRIMARY KEY,
    post_id UUID NOT NULL UNIQUE REFERENCES posts (id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size BIGINT NOT NULL,
    data BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT post_attachments_size_check CHECK (size > 0 AND size <= 10485760)
);

CREATE INDEX IF NOT EXISTS idx_post_attachments_post_id ON post_attachments (post_id);
