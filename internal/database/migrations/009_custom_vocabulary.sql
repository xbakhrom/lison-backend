-- Words the learner picks up in conversation with Maks. They live outside the
-- authored catalogue but share the same spaced-repetition machinery, so
-- user_cards now points at exactly one of the two sources.

CREATE TABLE user_vocabulary_items (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
    russian    text NOT NULL CHECK (russian <> ''),
    uzbek      text NOT NULL CHECK (uzbek <> ''),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX user_vocabulary_items_word_idx
    ON user_vocabulary_items (user_id, lower(russian));

ALTER TABLE user_cards
    ADD COLUMN user_vocabulary_item_id bigint REFERENCES user_vocabulary_items(id) ON DELETE CASCADE,
    ALTER COLUMN vocabulary_item_id DROP NOT NULL,
    ADD CONSTRAINT user_cards_one_source
        CHECK (num_nonnulls(vocabulary_item_id, user_vocabulary_item_id) = 1);

CREATE UNIQUE INDEX user_cards_custom_item_idx
    ON user_cards (user_id, user_vocabulary_item_id)
    WHERE user_vocabulary_item_id IS NOT NULL;
