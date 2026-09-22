-- Voice assistant ("Maks"): daily usage accounting and the learner level the
-- session prompt is built from.

CREATE TABLE user_assistant_usage (
    user_id    bigint NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
    usage_date date NOT NULL,
    seconds    integer NOT NULL DEFAULT 0 CHECK (seconds >= 0),
    sessions   integer NOT NULL DEFAULT 0 CHECK (sessions >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, usage_date)
);

ALTER TABLE users
    ADD COLUMN russian_level text NOT NULL DEFAULT '';
