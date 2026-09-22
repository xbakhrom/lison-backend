-- The learning route is grouped into four curriculum stages. Topic rows, their
-- lessons and their question banks are owned by the authored files in
-- contents/grammar and rewritten by the content sync on every start, so this
-- migration only prepares the column they need.
ALTER TABLE grammar_topics
    ADD COLUMN IF NOT EXISTS stage text NOT NULL DEFAULT 'foundation'
        CHECK (stage IN ('foundation', 'verbs', 'cases', 'fluency'));

CREATE INDEX IF NOT EXISTS grammar_topics_route_idx ON grammar_topics(status, position);

-- The sync rewrites every topic in one transaction, so two topics may hold the
-- same slug between statements while a rename moves it. Deferring the check to
-- commit time keeps a legitimate rename from failing the whole start-up.
ALTER TABLE grammar_topics DROP CONSTRAINT IF EXISTS grammar_topics_slug_key;
ALTER TABLE grammar_topics
    ADD CONSTRAINT grammar_topics_slug_key UNIQUE (slug) DEFERRABLE INITIALLY IMMEDIATE;
