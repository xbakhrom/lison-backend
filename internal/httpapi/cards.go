package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type cardResponse struct {
	ID           int64   `json:"id"`
	VocabularyID string  `json:"vocabularyId"`
	Russian      string  `json:"russian"`
	Uzbek        string  `json:"uzbek"`
	TopicTitle   string  `json:"topicTitle"`
	State        string  `json:"state"`
	IntervalDays int     `json:"intervalDays"`
	EaseFactor   float64 `json:"easeFactor"`
	DueDate      string  `json:"dueDate"`
	Repetitions  int     `json:"repetitions"`
	Lapses       int     `json:"lapses"`
}

func (s *Server) addCards(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	var request struct {
		VocabularyItemIDs []string `json:"vocabularyItemIds"`
	}
	if err := decodeJSON(w, r, &request); err != nil || len(request.VocabularyItemIDs) == 0 || len(request.VocabularyItemIDs) > 500 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Выберите от 1 до 500 слов.")
		return
	}
	dueDate := localToday(r)
	rows, err := s.db.Query(r.Context(), `
		INSERT INTO user_cards (user_id, vocabulary_item_id, due_date)
		SELECT $1, id, $3 FROM vocabulary_items
		WHERE id = ANY($2::text[]) AND active
		ON CONFLICT (user_id, vocabulary_item_id) DO NOTHING
		RETURNING id`, user.ID, request.VocabularyItemIDs, dueDate)
	if err != nil {
		s.logger.Error("add cards", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось добавить карточки.")
		return
	}
	defer rows.Close()
	added := 0
	for rows.Next() {
		added++
	}
	writeJSON(w, http.StatusCreated, map[string]int{"added": added})
}

func (s *Server) listCards(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	rows, err := s.db.Query(r.Context(), `
		SELECT c.id, v.id, v.russian, v.uzbek, t.title, c.state,
		       c.interval_days, c.ease_factor::float8, c.due_date::text,
		       c.repetitions, c.lapses
		FROM user_cards c
		JOIN vocabulary_items v ON v.id = c.vocabulary_item_id
		JOIN topics t ON t.id = v.topic_id
		WHERE c.user_id = $1
		ORDER BY c.due_date, c.created_at`, user.ID)
	if err != nil {
		s.logger.Error("list cards", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось загрузить карточки.")
		return
	}
	defer rows.Close()
	cards := make([]cardResponse, 0)
	for rows.Next() {
		var card cardResponse
		if err := rows.Scan(&card.ID, &card.VocabularyID, &card.Russian, &card.Uzbek, &card.TopicTitle, &card.State, &card.IntervalDays, &card.EaseFactor, &card.DueDate, &card.Repetitions, &card.Lapses); err != nil {
			s.logger.Error("scan card", "error", err)
			writeError(w, http.StatusInternalServerError, "database_error", "Не удалось загрузить карточки.")
			return
		}
		cards = append(cards, card)
	}
	writeJSON(w, http.StatusOK, map[string]any{"cards": cards, "dueCount": countDue(cards, localToday(r))})
}

func countDue(cards []cardResponse, today time.Time) int {
	count := 0
	date := today.Format("2006-01-02")
	for _, card := range cards {
		if card.DueDate <= date {
			count++
		}
	}
	return count
}

func (s *Server) deleteCard(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	cardID, err := strconv.ParseInt(chi.URLParam(r, "cardID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_card", "Некорректная карточка.")
		return
	}
	result, err := s.db.Exec(r.Context(), "DELETE FROM user_cards WHERE id = $1 AND user_id = $2", cardID, user.ID)
	if err != nil {
		s.logger.Error("delete card", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось удалить карточку.")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "card_not_found", "Карточка не найдена.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func localToday(r *http.Request) time.Time {
	zone := r.Header.Get("X-Timezone")
	location, err := time.LoadLocation(zone)
	if err != nil {
		location, _ = time.LoadLocation("Asia/Tashkent")
	}
	now := time.Now().In(location)
	year, month, day := now.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, location)
}
