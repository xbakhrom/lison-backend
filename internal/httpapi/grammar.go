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
	Stage        string `json:"stage"`
	Status       string `json:"status"`
	BestScore    int    `json:"bestScore"`
	DueDate      string `json:"dueDate,omitempty"`
	Due          bool   `json:"due"`
	LessonBlocks int    `json:"lessonBlocks"`
	MasteryLevel int    `json:"masteryLevel"`
}

type grammarProgressResponse struct {
	Status       string `json:"status"`
	BestScore    int    `json:"bestScore"`
	DueDate      string `json:"dueDate,omitempty"`
	Repetitions  int    `json:"repetitions"`
	MasteryLevel int    `json:"masteryLevel"`
}

type grammarTopicResponse struct {
	ID       string                  `json:"id"`
	Slug     string                  `json:"slug"`
	Title    string                  `json:"title"`
	Summary  string                  `json:"summary"`
	Level    string                  `json:"level"`
	Icon     string                  `json:"icon"`
	Stage    string                  `json:"stage"`
	Lesson   json.RawMessage         `json:"lesson"`
	Practice json.RawMessage         `json:"practice"`
	Game     json.RawMessage         `json:"game"`
	Progress grammarProgressResponse `json:"progress"`
}

type grammarQuestion struct {
	ID         string `json:"id"`
	Answer     string `json:"answer"`
	Difficulty int    `json:"difficulty"`
}

type submittedGrammarAnswer struct {
	QuestionID string `json:"questionId"`
	Answer     string `json:"answer"`
}

func (s *Server) listGrammarTopics(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	today := localToday(r)
	rows, err := s.db.Query(r.Context(), `
		SELECT t.id, t.slug, t.title, t.summary, t.level, t.icon, t.stage,
		       COALESCE(p.status, 'new'), COALESCE(p.best_score, 0),
		       COALESCE(p.due_date::text, ''),
		       COALESCE(p.due_date <= $2 AND p.status IN ('learning', 'review'), false),
		       jsonb_array_length(t.lesson), COALESCE(p.mastery_level, 1)
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
		if err := rows.Scan(&topic.ID, &topic.Slug, &topic.Title, &topic.Summary, &topic.Level, &topic.Icon, &topic.Stage,
			&topic.Status, &topic.BestScore, &topic.DueDate, &topic.Due, &topic.LessonBlocks, &topic.MasteryLevel); err != nil {
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
		SELECT t.id, t.slug, t.title, t.summary, t.level, t.icon, t.stage,
		       t.lesson, t.practice, t.game,
		       COALESCE(p.status, 'new'), COALESCE(p.best_score, 0),
		       COALESCE(p.due_date::text, ''), COALESCE(p.repetitions, 0),
		       COALESCE(p.mastery_level, 1)
		FROM grammar_topics t
		LEFT JOIN user_grammar_progress p ON p.topic_id = t.id AND p.user_id = $2
		WHERE t.slug = $1 AND t.status = 'published'`, chi.URLParam(r, "slug"), user.ID,
	).Scan(&topic.ID, &topic.Slug, &topic.Title, &topic.Summary, &topic.Level, &topic.Icon, &topic.Stage,
		&topic.Lesson, &topic.Practice, &topic.Game,
		&topic.Progress.Status, &topic.Progress.BestScore, &topic.Progress.DueDate, &topic.Progress.Repetitions,
		&topic.Progress.MasteryLevel)
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
		Answers []submittedGrammarAnswer `json:"answers"`
	}
	if err := decodeJSON(w, r, &request); err != nil || len(request.Answers) < 4 || len(request.Answers) > 12 {
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

	questionByID := make(map[string]grammarQuestion, len(questions))
	for _, question := range questions {
		if question.Difficulty < grammar.MinDifficulty || question.Difficulty > grammar.MaxDifficulty {
			question.Difficulty = grammar.MinDifficulty
		}
		questionByID[question.ID] = question
	}

	correct := 0
	seen := make(map[string]bool, len(request.Answers))
	checked := make([]bool, 0, len(request.Answers))
	for _, submitted := range request.Answers {
		question, exists := questionByID[submitted.QuestionID]
		if !exists || seen[submitted.QuestionID] {
			writeError(w, http.StatusBadRequest, "invalid_answers", "В ответах есть неизвестный или повторяющийся вопрос.")
			return
		}
		seen[submitted.QuestionID] = true
		isCorrect := submitted.Answer == question.Answer
		if isCorrect {
			correct++
		}
		checked = append(checked, isCorrect)
	}
	score := correct * 100 / len(request.Answers)

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить результат.")
		return
	}
	defer tx.Rollback(r.Context())

	var progress grammar.Progress
	masteryLevel := grammar.MinDifficulty
	err = tx.QueryRow(r.Context(), `
		SELECT interval_days, ease_factor::float8, repetitions, mastery_level
		FROM user_grammar_progress WHERE user_id = $1 AND topic_id = $2 FOR UPDATE`, user.ID, topicID,
	).Scan(&progress.IntervalDays, &progress.EaseFactor, &progress.Repetitions, &masteryLevel)
	if err != nil {
		progress = grammar.Progress{}
		masteryLevel = grammar.MinDifficulty
	}
	adaptive := grammar.AdaptiveState{Difficulty: masteryLevel}
	for _, isCorrect := range checked {
		adaptive = grammar.AdvanceDifficulty(adaptive, isCorrect)
	}
	schedule, err := grammar.Schedule(progress, score, localToday(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_score", "Некорректный результат игры.")
		return
	}

	_, err = tx.Exec(r.Context(), `
		INSERT INTO user_grammar_progress
			(user_id, topic_id, status, best_score, interval_days, ease_factor, repetitions, due_date, mastery_level, last_reviewed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		ON CONFLICT (user_id, topic_id) DO UPDATE SET
			status = EXCLUDED.status,
			best_score = GREATEST(user_grammar_progress.best_score, EXCLUDED.best_score),
			interval_days = EXCLUDED.interval_days,
			ease_factor = EXCLUDED.ease_factor,
			repetitions = EXCLUDED.repetitions,
			due_date = EXCLUDED.due_date,
			mastery_level = EXCLUDED.mastery_level,
			last_reviewed_at = now(), updated_at = now()`,
		user.ID, topicID, schedule.Status, score, schedule.IntervalDays, schedule.EaseFactor,
		schedule.Repetitions, schedule.DueDate, adaptive.Difficulty)
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO grammar_review_logs (user_id, topic_id, score, correct_count, question_count, next_interval, mastery_level)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			user.ID, topicID, score, correct, len(request.Answers), schedule.IntervalDays, adaptive.Difficulty)
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
		"score": score, "correct": correct, "total": len(request.Answers),
		"status": schedule.Status, "intervalDays": schedule.IntervalDays,
		"dueDate": schedule.DueDate.Format("2006-01-02"), "masteryLevel": adaptive.Difficulty,
	})
}
