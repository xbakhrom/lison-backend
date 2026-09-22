CREATE TABLE feedback_submissions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
    category text NOT NULL CHECK (category IN ('idea', 'bug', 'content', 'other')),
    message text NOT NULL CHECK (char_length(message) BETWEEN 5 AND 1000),
    screen text NOT NULL DEFAULT '',
    topic_slug text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'read', 'closed')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX feedback_submissions_user_created_idx
    ON feedback_submissions(user_id, created_at DESC);
