package discussion

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Sync publishes the embedded question sets into discussion_questions. Questions
// that disappear from the bank are deactivated rather than deleted, so a
// learner's discussion log keeps pointing at something real.
func Sync(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	sets, err := LoadSets()
	if err != nil {
		return 0, err
	}
	if issues := Validate(sets); len(issues) > 0 {
		return 0, fmt.Errorf("discussion content is invalid: %s (and %d more)", issues[0], len(issues)-1)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin discussion sync: %w", err)
	}
	defer tx.Rollback(ctx)

	ids := make([]string, 0)
	for _, set := range sets {
		// A set may name a topic that is not published yet; store it as general
		// rather than failing the whole sync on a foreign key.
		var topicID *string
		if set.TopicID != "" {
			var exists bool
			if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM topics WHERE id = $1)", set.TopicID).Scan(&exists); err != nil {
				return 0, fmt.Errorf("check topic %s: %w", set.TopicID, err)
			}
			if exists {
				topicID = &set.TopicID
			}
		}

		for position, question := range set.Questions {
			followUps, err := json.Marshal(question.FollowUps)
			if err != nil {
				return 0, fmt.Errorf("encode follow-ups of %s: %w", question.ID, err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO discussion_questions
					(id, set_id, topic_id, question, uzbek_hint, level, follow_ups, position, active)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)
				ON CONFLICT (id) DO UPDATE SET
					set_id = EXCLUDED.set_id,
					topic_id = EXCLUDED.topic_id,
					question = EXCLUDED.question,
					uzbek_hint = EXCLUDED.uzbek_hint,
					level = EXCLUDED.level,
					follow_ups = EXCLUDED.follow_ups,
					position = EXCLUDED.position,
					active = true`,
				question.ID, set.ID, topicID, question.Question, question.UzbekHint,
				question.Level, followUps, position+1); err != nil {
				return 0, fmt.Errorf("upsert discussion question %s: %w", question.ID, err)
			}
			ids = append(ids, question.ID)
		}
	}

	if _, err := tx.Exec(ctx,
		"UPDATE discussion_questions SET active = false WHERE id <> ALL($1) AND active", ids); err != nil {
		return 0, fmt.Errorf("retire discussion questions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit discussion sync: %w", err)
	}
	return len(ids), nil
}
