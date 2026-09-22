package reminder

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xbakhrom/lison-backend/internal/telegram"
)

type Worker struct {
	db       *pgxpool.Pool
	telegram *telegram.Client
	logger   *slog.Logger
}

func New(db *pgxpool.Pool, telegramClient *telegram.Client, logger *slog.Logger) *Worker {
	return &Worker{db: db, telegram: telegramClient, logger: logger}
}

func (w *Worker) Run(ctx context.Context) {
	if !w.telegram.Enabled() {
		w.logger.Info("reminder worker disabled: Telegram credentials are not configured")
		return
	}
	w.tick(ctx)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	w.process(ctx)
	w.processDiscussions(ctx)
}

// processDiscussions sends the question of the day to learners who have nothing
// left to review, so the daily nudge still has something to offer them. Learners
// with due cards get the review reminder instead; one push a day is enough.
func (w *Worker) processDiscussions(ctx context.Context) {
	rows, err := w.db.Query(ctx, `
		SELECT r.user_id, q.id, q.question, q.uzbek_hint,
		       (now() AT TIME ZONE r.timezone)::date AS local_date
		FROM reminder_settings r
		CROSS JOIN LATERAL (
			SELECT d.id, d.question, d.uzbek_hint
			FROM discussion_questions d
			LEFT JOIN user_discussion_log l ON l.question_id = d.id AND l.user_id = r.user_id
			WHERE d.active AND l.question_id IS NULL
			ORDER BY d.position, d.id
			LIMIT 1
		) q
		WHERE r.enabled
		  AND r.reminder_time <= (now() AT TIME ZONE r.timezone)::time
		  AND (r.last_discussion_on IS NULL OR r.last_discussion_on < (now() AT TIME ZONE r.timezone)::date)
		  AND NOT EXISTS (
		      SELECT 1 FROM user_cards c WHERE c.user_id = r.user_id
		      AND c.due_date <= (now() AT TIME ZONE r.timezone)::date)
		  AND NOT EXISTS (
		      SELECT 1 FROM user_grammar_progress g WHERE g.user_id = r.user_id
		      AND g.status IN ('learning', 'review')
		      AND g.due_date <= (now() AT TIME ZONE r.timezone)::date)
		ORDER BY r.user_id
		LIMIT 100`)
	if err != nil {
		w.logger.Error("find discussion nudges", "error", err)
		return
	}
	defer rows.Close()
	type candidate struct {
		userID     int64
		questionID string
		question   string
		uzbekHint  string
		localDate  time.Time
	}
	candidates := make([]candidate, 0)
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.userID, &item.questionID, &item.question, &item.uzbekHint, &item.localDate); err != nil {
			w.logger.Error("scan discussion nudge", "error", err)
			return
		}
		candidates = append(candidates, item)
	}
	for _, item := range candidates {
		if err := w.telegram.SendDiscussion(ctx, item.userID, item.question, item.uzbekHint); err != nil {
			w.logger.Error("send discussion nudge", "user_id", item.userID, "error", err)
			continue
		}
		if _, err := w.db.Exec(ctx, `
			UPDATE reminder_settings SET last_discussion_on = $2, updated_at = now()
			WHERE user_id = $1`, item.userID, item.localDate); err != nil {
			w.logger.Error("mark discussion nudge sent", "user_id", item.userID, "error", err)
		}
	}
}

func (w *Worker) process(ctx context.Context) {
	rows, err := w.db.Query(ctx, `
		SELECT r.user_id,
		       (SELECT count(*)::int FROM user_cards c
		        WHERE c.user_id = r.user_id
		          AND c.due_date <= (now() AT TIME ZONE r.timezone)::date) AS card_count,
		       (SELECT count(*)::int FROM user_grammar_progress g
		        WHERE g.user_id = r.user_id
		          AND g.status IN ('learning', 'review')
		          AND g.due_date <= (now() AT TIME ZONE r.timezone)::date) AS grammar_count,
		       (now() AT TIME ZONE r.timezone)::date AS local_date
		FROM reminder_settings r
		WHERE r.enabled
		  AND ((SELECT count(*) FROM user_cards c WHERE c.user_id = r.user_id
		        AND c.due_date <= (now() AT TIME ZONE r.timezone)::date) > 0
		    OR (SELECT count(*) FROM user_grammar_progress g WHERE g.user_id = r.user_id
		        AND g.status IN ('learning', 'review')
		        AND g.due_date <= (now() AT TIME ZONE r.timezone)::date) > 0)
		  AND r.reminder_time <= (now() AT TIME ZONE r.timezone)::time
		  AND (r.last_sent_on IS NULL OR r.last_sent_on < (now() AT TIME ZONE r.timezone)::date)
		ORDER BY r.user_id
		LIMIT 100`)
	if err != nil {
		w.logger.Error("find due reminders", "error", err)
		return
	}
	defer rows.Close()
	type candidate struct {
		userID       int64
		cardCount    int
		grammarCount int
		localDate    time.Time
	}
	candidates := make([]candidate, 0)
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.userID, &item.cardCount, &item.grammarCount, &item.localDate); err != nil {
			w.logger.Error("scan reminder", "error", err)
			return
		}
		candidates = append(candidates, item)
	}
	for _, item := range candidates {
		if err := w.telegram.SendReminder(ctx, item.userID, item.cardCount, item.grammarCount); err != nil {
			w.logger.Error("send reminder", "user_id", item.userID, "error", err)
			continue
		}
		if _, err := w.db.Exec(ctx, `
			UPDATE reminder_settings SET last_sent_on = $2, updated_at = now()
			WHERE user_id = $1`, item.userID, item.localDate); err != nil {
			w.logger.Error("mark reminder sent", "user_id", item.userID, "error", err)
		}
	}
}
