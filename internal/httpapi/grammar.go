package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xbakhrom/lison-backend/internal/grammar"
)

type grammarListItem struct {
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	Level        string `json:"level"`
	Icon         string `json:"icon"`
	Status       string `json:"status"`
	BestScore    int    `json:"bestScore"`
	DueDate      string `json:"dueDate,omitempty"`
	Due          bool   `json:"due"`
	LessonBlocks int    `json:"lessonBlocks"`
}

type grammarProgressResponse struct {
	Status      string `json:"status"`
	BestScore   int    `json:"bestScore"`
	DueDate     string `json:"dueDate,omitempty"`
	Repetitions int    `json:"repetitions"`
}

type grammarTopicResponse struct {
	ID       string                  `json:"id"`
	Slug     string                  `json:"slug"`
	Title    string                  `json:"title"`
	Summary  string                  `json:"summary"`
	Level    string                  `json:"level"`
	Icon     string                  `json:"icon"`
	Lesson   json.RawMessage         `json:"lesson"`
	Practice json.RawMessage         `json:"practice"`
	Game     json.RawMessage         `json:"game"`
	Progress grammarProgressResponse `json:"progress"`
}

type grammarQuestion struct {
	ID     string `json:"id"`
	Answer string `json:"answer"`
}

func (s *Server) listGrammarTopics(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	today := localToday(r)
	rows, err := s.db.Query(r.Context(), `
		SELECT t.id, t.slug, t.title, t.summary, t.level, t.icon,
		       COALESCE(p.status, 'new'), COALESCE(p.best_score, 0),
		       COALESCE(p.due_date::text, ''),
		       COALESCE(p.due_date <= $2 AND p.status IN ('learning', 'review'), false),
		       jsonb_array_length(t.lesson)
		FROM grammar_topics t
		LEFT JOIN user_grammar_progress p ON p.topic_id = t.id AND p.user_id = $1
		WHERE t.status = 'published'
		ORDER BY t.position`, user.ID, today)
	if err != nil {
		s.logger.Error("list grammar topics", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось загрузить грамматику.")
		return
	}
	defer rows.Close()

	topics := make([]grammarListItem, 0)
	dueCount := 0
	for rows.Next() {
		var topic grammarListItem
		if err := rows.Scan(&topic.ID, &topic.Slug, &topic.Title, &topic.Summary, &topic.Level, &topic.Icon,
			&topic.Status, &topic.BestScore, &topic.DueDate, &topic.Due, &topic.LessonBlocks); err != nil {
			writeError(w, http.StatusInternalServerError, "database_error", "Не удалось загрузить грамматику.")
			return
		}
		if topic.Due {
			dueCount++
		}
		topics = append(topics, topic)
	}
	if err := rows.Err(); err != nil {
		s.logger.Error("iterate grammar topics", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось загрузить грамматику.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"topics": topics, "dueCount": dueCount})
}

func (s *Server) getGrammarTopic(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	var topic grammarTopicResponse
	err := s.db.QueryRow(r.Context(), `
		SELECT t.id, t.slug, t.title, t.summary, t.level, t.icon,
		       t.lesson, t.practice, t.game,
		       COALESCE(p.status, 'new'), COALESCE(p.best_score, 0),
		       COALESCE(p.due_date::text, ''), COALESCE(p.repetitions, 0)
		FROM grammar_topics t
		LEFT JOIN user_grammar_progress p ON p.topic_id = t.id AND p.user_id = $2
		WHERE t.slug = $1 AND t.status = 'published'`, chi.URLParam(r, "slug"), user.ID,
	).Scan(&topic.ID, &topic.Slug, &topic.Title, &topic.Summary, &topic.Level, &topic.Icon,
		&topic.Lesson, &topic.Practice, &topic.Game,
		&topic.Progress.Status, &topic.Progress.BestScore, &topic.Progress.DueDate, &topic.Progress.Repetitions)
	if err != nil {
		writeError(w, http.StatusNotFound, "grammar_topic_not_found", "Грамматическая тема не найдена.")
		return
	}
	writeJSON(w, http.StatusOK, topic)
}

func (s *Server) finishGrammarGame(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	topicID := chi.URLParam(r, "topicID")
	var request struct {
		Answers map[string]string `json:"answers"`
	}
	if err := decodeJSON(w, r, &request); err != nil || len(request.Answers) == 0 || len(request.Answers) > 20 {
		writeError(w, http.StatusBadRequest, "invalid_answers", "Завершите игру и отправьте ответы.")
		return
	}

	var rawQuestions []byte
	if err := s.db.QueryRow(r.Context(), "SELECT game FROM grammar_topics WHERE id = $1 AND status = 'published'", topicID).Scan(&rawQuestions); err != nil {
		writeError(w, http.StatusNotFound, "grammar_topic_not_found", "Грамматическая тема не найдена.")
		return
	}
	var questions []grammarQuestion
	if err := json.Unmarshal(rawQuestions, &questions); err != nil || len(questions) == 0 {
		s.logger.Error("decode grammar game", "topic_id", topicID, "error", err)
		writeError(w, http.StatusInternalServerError, "content_error", "Не удалось проверить игру.")
		return
	}

	correct := 0
	for _, question := range questions {
		if request.Answers[question.ID] == question.Answer {
			correct++
		}
	}
	score := correct * 100 / len(questions)

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить результат.")
		return
	}
	defer tx.Rollback(r.Context())

	var progress grammar.Progress
	err = tx.QueryRow(r.Context(), `
		SELECT interval_days, ease_factor::float8, repetitions
		FROM user_grammar_progress WHERE user_id = $1 AND topic_id = $2 FOR UPDATE`, user.ID, topicID,
	).Scan(&progress.IntervalDays, &progress.EaseFactor, &progress.Repetitions)
	if err != nil {
		progress = grammar.Progress{}
	}
	schedule, err := grammar.Schedule(progress, score, localToday(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_score", "Некорректный результат игры.")
		return
	}

	_, err = tx.Exec(r.Context(), `
		INSERT INTO user_grammar_progress
			(user_id, topic_id, status, best_score, interval_days, ease_factor, repetitions, due_date, last_reviewed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		ON CONFLICT (user_id, topic_id) DO UPDATE SET
			status = EXCLUDED.status,
			best_score = GREATEST(user_grammar_progress.best_score, EXCLUDED.best_score),
			interval_days = EXCLUDED.interval_days,
			ease_factor = EXCLUDED.ease_factor,
			repetitions = EXCLUDED.repetitions,
			due_date = EXCLUDED.due_date,
			last_reviewed_at = now(), updated_at = now()`,
		user.ID, topicID, schedule.Status, score, schedule.IntervalDays, schedule.EaseFactor,
		schedule.Repetitions, schedule.DueDate)
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO grammar_review_logs (user_id, topic_id, score, correct_count, question_count, next_interval)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			user.ID, topicID, score, correct, len(questions), schedule.IntervalDays)
	}
	if err != nil {
		s.logger.Error("save grammar result", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить результат.")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить результат.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"score": score, "correct": correct, "total": len(questions),
		"status": schedule.Status, "intervalDays": schedule.IntervalDays,
		"dueDate": schedule.DueDate.Format("2006-01-02"),
	})
}
