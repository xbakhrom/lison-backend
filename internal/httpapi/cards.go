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

// customCardsTitle groups words the learner added themselves, which have no
// authored topic behind them.
const customCardsTitle = "Мои слова"

// cardSelect reads a card from either source: the authored catalogue or the
// learner's own words. Callers append their own WHERE/ORDER BY.
const cardSelect = `
	SELECT c.id,
	       COALESCE(v.id, 'custom:' || uv.id::text),
	       COALESCE(v.russian, uv.russian),
	       COALESCE(v.uzbek, uv.uzbek),
	       COALESCE(t.title, '` + customCardsTitle + `'),
	       c.state, c.interval_days, c.ease_factor::float8, c.due_date::text,
	       c.repetitions, c.lapses
	FROM user_cards c
	LEFT JOIN vocabulary_items v ON v.id = c.vocabulary_item_id
	LEFT JOIN topics t ON t.id = v.topic_id
	LEFT JOIN user_vocabulary_items uv ON uv.id = c.user_vocabulary_item_id`

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
	rows, err := s.db.Query(r.Context(), cardSelect+`
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
	// Deleting a card built on the learner's own word drops the word too: it has
	// no home outside the card.
	var customItemID *int64
	err = s.db.QueryRow(r.Context(), `
		DELETE FROM user_cards WHERE id = $1 AND user_id = $2
		RETURNING user_vocabulary_item_id`, cardID, user.ID).Scan(&customItemID)
	if isNoRows(err) {
		writeError(w, http.StatusNotFound, "card_not_found", "Карточка не найдена.")
		return
	}
	if err != nil {
		s.logger.Error("delete card", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось удалить карточку.")
		return
	}
	if customItemID != nil {
		if _, err := s.db.Exec(r.Context(),
			"DELETE FROM user_vocabulary_items WHERE id = $1 AND user_id = $2", *customItemID, user.ID); err != nil {
			s.logger.Error("delete custom word", "error", err)
		}
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
