-- Conversation questions Maks pulls from, and the record of what each learner
-- has already talked through.

CREATE TABLE discussion_questions (
    id         text PRIMARY KEY,
    set_id     text NOT NULL,
    topic_id   text REFERENCES topics(id) ON DELETE SET NULL,
    question   text NOT NULL,
    uzbek_hint text NOT NULL DEFAULT '',
    level      text NOT NULL DEFAULT '',
    follow_ups jsonb NOT NULL DEFAULT '[]'::jsonb,
    position   integer NOT NULL DEFAULT 0,
    active     boolean NOT NULL DEFAULT true
);

CREATE INDEX discussion_questions_topic_idx ON discussion_questions (topic_id, position) WHERE active;

CREATE TABLE user_discussion_log (
    user_id      bigint NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
    question_id  text NOT NULL REFERENCES discussion_questions(id) ON DELETE CASCADE,
    summary      text NOT NULL DEFAULT '',
    discussed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, question_id)
);

CREATE INDEX user_discussion_log_recent_idx ON user_discussion_log (user_id, discussed_at DESC);

-- Which discussion question was pushed on which local day, so the daily nudge
-- fires once and does not repeat a question the learner already saw.
ALTER TABLE reminder_settings
    ADD COLUMN last_discussion_on date;
