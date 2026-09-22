package grammar

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Sync publishes the embedded content bank into grammar_topics. The authored
// files are the single source of truth: every run rewrites lessons, banks and
// curriculum order. Topics that disappear from the bank are archived instead of
// deleted so a learner keeps the progress rows that point at them.
func Sync(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	topics, err := LoadTopics()
	if err != nil {
		return 0, err
	}
	if issues := Validate(topics); len(issues) > 0 {
		return 0, fmt.Errorf("grammar content is invalid: %s (and %d more)", issues[0], len(issues)-1)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin grammar sync: %w", err)
	}
	defer tx.Rollback(ctx)

	// A renamed slug can briefly collide with the topic that still holds it, so
	// the unique check waits until every topic has been written.
	if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
		return 0, fmt.Errorf("defer grammar constraints: %w", err)
	}

	ids := make([]string, 0, len(topics))
	for _, topic := range topics {
		lesson, practice, game, err := encodeBanks(topic)
		if err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO grammar_topics
				(id, slug, title, summary, level, icon, stage, position, status, lesson, practice, game, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now())
			ON CONFLICT (id) DO UPDATE SET
				slug = EXCLUDED.slug,
				title = EXCLUDED.title,
				summary = EXCLUDED.summary,
				level = EXCLUDED.level,
				icon = EXCLUDED.icon,
				stage = EXCLUDED.stage,
				position = EXCLUDED.position,
				status = EXCLUDED.status,
				lesson = EXCLUDED.lesson,
				practice = EXCLUDED.practice,
				game = EXCLUDED.game,
				updated_at = now()`,
			topic.ID, topic.Slug, topic.Title, topic.Summary, topic.Level, topic.Icon, topic.Stage,
			topic.Position, topic.Status, lesson, practice, game); err != nil {
			return 0, fmt.Errorf("upsert grammar topic %s: %w", topic.ID, err)
		}
		ids = append(ids, topic.ID)
	}

	if _, err := tx.Exec(ctx,
		"UPDATE grammar_topics SET status = 'archived', updated_at = now() WHERE id <> ALL($1) AND status <> 'archived'",
		ids); err != nil {
		return 0, fmt.Errorf("archive retired grammar topics: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit grammar sync: %w", err)
	}
	return len(topics), nil
}

func encodeBanks(topic Topic) (lesson, practice, game []byte, err error) {
	if lesson, err = json.Marshal(topic.Lesson); err != nil {
		return nil, nil, nil, fmt.Errorf("encode lesson of %s: %w", topic.ID, err)
	}
	if practice, err = json.Marshal(topic.Practice); err != nil {
		return nil, nil, nil, fmt.Errorf("encode practice of %s: %w", topic.ID, err)
	}
	if game, err = json.Marshal(topic.Game); err != nil {
		return nil, nil, nil, fmt.Errorf("encode game of %s: %w", topic.ID, err)
	}
	return lesson, practice, game, nil
}
