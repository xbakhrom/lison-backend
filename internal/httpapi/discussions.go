package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

const discussionSummaryMaxRunes = 1000

type discussionQuestion struct {
	ID        string   `json:"id"`
	TopicID   string   `json:"topicId"`
	Question  string   `json:"question"`
	UzbekHint string   `json:"uzbekHint"`
	Level     string   `json:"level"`
	FollowUps []string `json:"followUps"`
}

// nextDiscussion picks a question the learner has not talked through yet. With a
// topicId it stays inside that lesson; without one it prefers lesson questions
// from topics the learner is already studying, then falls back to small talk.
func (s *Server) nextDiscussion(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	topicID := strings.TrimSpace(r.URL.Query().Get("topicId"))

	var question discussionQuestion
	var followUps []byte
	var topic *string
	err := s.db.QueryRow(r.Context(), `
		SELECT q.id, q.topic_id, q.question, q.uzbek_hint, q.level, q.follow_ups
		FROM discussion_questions q
		LEFT JOIN user_discussion_log l ON l.question_id = q.id AND l.user_id = $1
		WHERE q.active
		  AND l.question_id IS NULL
		  AND ($2 = '' OR q.topic_id = $2)
		ORDER BY
			-- Without a topic filter, lead with lessons the learner already has
			-- cards from, so the question lands on familiar vocabulary.
			CASE WHEN q.topic_id IS NULL THEN 1 ELSE 0 END,
			CASE WHEN EXISTS (
				SELECT 1 FROM user_cards c
				JOIN vocabulary_items v ON v.id = c.vocabulary_item_id
				WHERE c.user_id = $1 AND v.topic_id = q.topic_id
			) THEN 0 ELSE 1 END,
			q.position, q.id
		LIMIT 1`, user.ID, topicID,
	).Scan(&question.ID, &topic, &question.Question, &question.UzbekHint, &question.Level, &followUps)
	if isNoRows(err) {
		// Everything has been covered. Not an error: Maks carries on by itself.
		writeJSON(w, http.StatusOK, map[string]any{"question": nil, "exhausted": true})
		return
	}
	if err != nil {
		s.logger.Error("next discussion", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось подобрать вопрос.")
		return
	}
	if topic != nil {
		question.TopicID = *topic
	}
	if err := json.Unmarshal(followUps, &question.FollowUps); err != nil {
		question.FollowUps = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"question": question, "exhausted": false})
}

func (s *Server) logDiscussion(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	questionID := chi.URLParam(r, "questionID")
	var request struct {
		Summary string `json:"summary"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Некорректный ответ.")
		return
	}
	summary := strings.TrimSpace(request.Summary)
	if runes := []rune(summary); len(runes) > discussionSummaryMaxRunes {
		summary = string(runes[:discussionSummaryMaxRunes])
	}

	tag, err := s.db.Exec(r.Context(), `
		INSERT INTO user_discussion_log (user_id, question_id, summary)
		SELECT $1, $2, $3 FROM discussion_questions WHERE id = $2
		ON CONFLICT (user_id, question_id) DO UPDATE
		SET summary = EXCLUDED.summary, discussed_at = now()`, user.ID, questionID, summary)
	if err != nil {
		s.logger.Error("log discussion", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить разговор.")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "question_not_found", "Вопрос не найден.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"saved": true})
}
