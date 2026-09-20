package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) saveAnswer(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	topicID := chi.URLParam(r, "topicID")
	fieldID := chi.URLParam(r, "fieldID")
	var request struct {
		Value json.RawMessage `json:"value"`
	}
	if err := decodeJSON(w, r, &request); err != nil || !json.Valid(request.Value) || len(request.Value) > 20_000 {
		writeError(w, http.StatusBadRequest, "invalid_value", "Проверьте введённый текст.")
		return
	}

	var inputType string
	err := s.db.QueryRow(r.Context(),
		"SELECT input_type FROM topic_form_fields WHERE id = $1 AND topic_id = $2", fieldID, topicID,
	).Scan(&inputType)
	if err != nil {
		writeError(w, http.StatusNotFound, "field_not_found", "Поле не найдено.")
		return
	}
	if inputType == "collection" && len(request.Value) > 0 && request.Value[0] != '[' {
		writeError(w, http.StatusBadRequest, "invalid_value", "Таблица должна содержать список строк.")
		return
	}
	if inputType != "collection" && len(request.Value) > 0 && request.Value[0] != '"' {
		writeError(w, http.StatusBadRequest, "invalid_value", "Поле должно содержать текст.")
		return
	}
	request.Value = bytes.TrimSpace(request.Value)
	_, err = s.db.Exec(r.Context(), `
		INSERT INTO user_topic_answers (user_id, topic_id, field_id, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, field_id) DO UPDATE
		SET value = EXCLUDED.value, updated_at = now()`, user.ID, topicID, fieldID, request.Value)
	if err != nil {
		s.logger.Error("save answer", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить ответ.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"saved": true})
}

func (s *Server) saveChecklist(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	topicID := chi.URLParam(r, "topicID")
	itemID := chi.URLParam(r, "itemID")
	var request struct {
		Checked bool `json:"checked"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Некорректное значение.")
		return
	}
	result, err := s.db.Exec(r.Context(), `
		INSERT INTO user_checklist_progress (user_id, item_id, checked)
		SELECT $1, id, $4 FROM topic_checklist_items WHERE id = $2 AND topic_id = $3
		ON CONFLICT (user_id, item_id) DO UPDATE
		SET checked = EXCLUDED.checked, updated_at = now()`, user.ID, itemID, topicID, request.Checked)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить отметку.")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "item_not_found", "Вопрос не найден.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"saved": true})
}
