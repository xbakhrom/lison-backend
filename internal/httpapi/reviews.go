package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xbakhrom/lison-backend/internal/srs"
)

func (s *Server) listDueCards(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	today := localToday(r)
	rows, err := s.db.Query(r.Context(), `
		SELECT c.id, v.id, v.russian, v.uzbek, t.title, c.state,
		       c.interval_days, c.ease_factor::float8, c.due_date::text,
		       c.repetitions, c.lapses
		FROM user_cards c
		JOIN vocabulary_items v ON v.id = c.vocabulary_item_id
		JOIN topics t ON t.id = v.topic_id
		WHERE c.user_id = $1 AND c.due_date <= $2
		ORDER BY CASE WHEN c.state = 'new' THEN 2 ELSE 1 END, c.due_date, c.id`, user.ID, today)
	if err != nil {
		s.logger.Error("list due cards", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось загрузить повторение.")
		return
	}
	defer rows.Close()
	cards := make([]cardResponse, 0)
	for rows.Next() {
		var card cardResponse
		if err := rows.Scan(&card.ID, &card.VocabularyID, &card.Russian, &card.Uzbek, &card.TopicTitle, &card.State, &card.IntervalDays, &card.EaseFactor, &card.DueDate, &card.Repetitions, &card.Lapses); err != nil {
			writeError(w, http.StatusInternalServerError, "database_error", "Не удалось загрузить повторение.")
			return
		}
		cards = append(cards, card)
	}
	writeJSON(w, http.StatusOK, map[string]any{"cards": cards})
}

func (s *Server) reviewCard(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	cardID, err := strconv.ParseInt(chi.URLParam(r, "cardID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_card", "Некорректная карточка.")
		return
	}
	var request struct {
		Rating srs.Rating `json:"rating"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Выберите оценку ответа.")
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить ответ.")
		return
	}
	defer tx.Rollback(r.Context())

	var card srs.Card
	err = tx.QueryRow(r.Context(), `
		SELECT state, interval_days, ease_factor::float8, repetitions, lapses
		FROM user_cards WHERE id = $1 AND user_id = $2 FOR UPDATE`, cardID, user.ID,
	).Scan(&card.State, &card.IntervalDays, &card.EaseFactor, &card.Repetitions, &card.Lapses)
	if err != nil {
		writeError(w, http.StatusNotFound, "card_not_found", "Карточка не найдена.")
		return
	}
	schedule, err := srs.Schedule(card, request.Rating, localToday(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_rating", "Неизвестная оценка ответа.")
		return
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE user_cards SET state = $1, interval_days = $2, ease_factor = $3,
			due_date = $4, repetitions = $5, lapses = $6, last_reviewed_at = now()
		WHERE id = $7`, schedule.State, schedule.IntervalDays, schedule.EaseFactor,
		schedule.DueDate, schedule.Repetitions, schedule.Lapses, cardID)
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO review_logs (user_id, card_id, rating, previous_interval, next_interval)
			VALUES ($1, $2, $3, $4, $5)`, user.ID, cardID, request.Rating, card.IntervalDays, schedule.IntervalDays)
	}
	if err != nil {
		s.logger.Error("save review", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить ответ.")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить ответ.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"state": schedule.State, "intervalDays": schedule.IntervalDays,
		"easeFactor": schedule.EaseFactor, "dueDate": schedule.DueDate.Format("2006-01-02"),
	})
}
