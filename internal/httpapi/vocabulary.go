package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	vocabularySearchDefaultLimit = 8
	vocabularySearchMaxLimit     = 25
)

// normalizedRussian folds a Russian expression down to what a speaker would
// actually say: catalogue words carry stress marks (Го́род) that nobody types or
// dictates, and ё/е are used interchangeably.
func normalizedRussian(expression string) string {
	return "translate(lower(" + expression + "), 'ё' || chr(769), 'е')"
}

type vocabularySearchResult struct {
	ID         string `json:"id"`
	Russian    string `json:"russian"`
	Uzbek      string `json:"uzbek"`
	TopicTitle string `json:"topicTitle"`
	Added      bool   `json:"added"`
}

// searchVocabulary maps a word Maks heard out loud onto a catalogue id, which is
// what the flashcard endpoints take.
func (s *Server) searchVocabulary(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Укажите слово для поиска.")
		return
	}
	if len([]rune(query)) > 64 {
		query = string([]rune(query)[:64])
	}

	limit := vocabularySearchDefaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_request", "Некорректное количество результатов.")
			return
		}
		limit = min(parsed, vocabularySearchMaxLimit)
	}

	// Exact matches first, then prefixes, then anything containing the query, so
	// the assistant can trust the first hit in the common case.
	rows, err := s.db.Query(r.Context(), `
		WITH needle AS (SELECT `+normalizedRussian("$1")+` AS q)
		SELECT v.id, v.russian, v.uzbek, t.title, uc.id IS NOT NULL
		FROM vocabulary_items v
		JOIN topics t ON t.id = v.topic_id
		CROSS JOIN needle n
		LEFT JOIN user_cards uc ON uc.vocabulary_item_id = v.id AND uc.user_id = $2
		WHERE v.active AND t.status = 'published'
		  AND (`+normalizedRussian("v.russian")+` LIKE '%' || n.q || '%'
		       OR lower(v.uzbek) LIKE '%' || n.q || '%')
		ORDER BY
			CASE WHEN `+normalizedRussian("v.russian")+` = n.q OR lower(v.uzbek) = n.q THEN 0
			     WHEN `+normalizedRussian("v.russian")+` LIKE n.q || '%' OR lower(v.uzbek) LIKE n.q || '%' THEN 1
			     ELSE 2 END,
			length(v.russian), v.russian
		LIMIT $3`, query, user.ID, limit)
	if err != nil {
		s.logger.Error("search vocabulary", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось найти слово.")
		return
	}
	defer rows.Close()

	results := make([]vocabularySearchResult, 0)
	for rows.Next() {
		var item vocabularySearchResult
		if err := rows.Scan(&item.ID, &item.Russian, &item.Uzbek, &item.TopicTitle, &item.Added); err != nil {
			s.logger.Error("scan vocabulary search", "error", err)
			writeError(w, http.StatusInternalServerError, "database_error", "Не удалось найти слово.")
			return
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		s.logger.Error("search vocabulary rows", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось найти слово.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

// addCustomWord stores a word that is not in the catalogue and gives it a
// flashcard, so anything picked up in conversation joins the review queue.
func (s *Server) addCustomWord(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	var request struct {
		Russian string `json:"russian"`
		Uzbek   string `json:"uzbek"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Некорректное слово.")
		return
	}
	russian := strings.TrimSpace(request.Russian)
	uzbek := strings.TrimSpace(request.Uzbek)
	if russian == "" || uzbek == "" || len([]rune(russian)) > 100 || len([]rune(uzbek)) > 100 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Укажите слово и перевод.")
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить слово.")
		return
	}
	defer tx.Rollback(r.Context())

	var itemID int64
	// Re-saying a word the learner already stored refreshes the translation
	// rather than erroring out mid-conversation.
	if err := tx.QueryRow(r.Context(), `
		INSERT INTO user_vocabulary_items (user_id, russian, uzbek)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, lower(russian)) DO UPDATE SET uzbek = EXCLUDED.uzbek
		RETURNING id`, user.ID, russian, uzbek).Scan(&itemID); err != nil {
		s.logger.Error("save custom word", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить слово.")
		return
	}

	var cardID int64
	var created bool
	err = tx.QueryRow(r.Context(), `
		WITH inserted AS (
			INSERT INTO user_cards (user_id, user_vocabulary_item_id, due_date)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, user_vocabulary_item_id)
				WHERE user_vocabulary_item_id IS NOT NULL DO NOTHING
			RETURNING id
		)
		SELECT id, true FROM inserted
		UNION ALL
		SELECT id, false FROM user_cards
		WHERE user_id = $1 AND user_vocabulary_item_id = $2
		LIMIT 1`, user.ID, itemID, localToday(r)).Scan(&cardID, &created)
	if err != nil {
		s.logger.Error("add custom card", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить слово.")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить слово.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"cardId":  cardID,
		"russian": russian,
		"uzbek":   uzbek,
		"created": created,
	})
}
