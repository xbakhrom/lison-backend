package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xbakhrom/lison-backend/internal/telegram"
)

var feedbackCategories = map[string]string{
	"idea":    "Идея",
	"bug":     "Ошибка",
	"content": "Контент",
	"other":   "Другое",
}

func (s *Server) submitFeedback(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	var request struct {
		Category  string `json:"category"`
		Message   string `json:"message"`
		Screen    string `json:"screen"`
		TopicSlug string `json:"topicSlug"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Проверьте заполненные поля.")
		return
	}

	request.Category = strings.TrimSpace(request.Category)
	request.Message = strings.TrimSpace(request.Message)
	request.Screen = strings.TrimSpace(request.Screen)
	request.TopicSlug = strings.TrimSpace(request.TopicSlug)
	categoryLabel, validCategory := feedbackCategories[request.Category]
	messageLength := utf8.RuneCountInString(request.Message)
	if !validCategory || messageLength < 5 || messageLength > 1000 ||
		utf8.RuneCountInString(request.Screen) > 32 || utf8.RuneCountInString(request.TopicSlug) > 100 {
		writeError(w, http.StatusBadRequest, "invalid_feedback", "Напишите предложение длиной от 5 до 1000 символов.")
		return
	}

	var recentCount int
	if err := s.db.QueryRow(r.Context(), `
		SELECT count(*) FROM feedback_submissions
		WHERE user_id = $1 AND created_at >= now() - interval '24 hours'`, user.ID,
	).Scan(&recentCount); err != nil {
		s.logger.Error("count feedback submissions", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось отправить предложение.")
		return
	}
	if recentCount >= 5 {
		writeError(w, http.StatusTooManyRequests, "feedback_limit", "Сегодня вы уже отправили несколько предложений. Попробуйте завтра.")
		return
	}

	if _, err := s.db.Exec(r.Context(), `
		INSERT INTO feedback_submissions (user_id, category, message, screen, topic_slug)
		VALUES ($1, $2, $3, $4, $5)`,
		user.ID, request.Category, request.Message, request.Screen, request.TopicSlug,
	); err != nil {
		s.logger.Error("save feedback", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось отправить предложение.")
		return
	}

	if s.cfg.FeedbackChatID != 0 && s.telegram.Enabled() {
		feedback := telegram.Feedback{
			UserID: user.ID, FirstName: user.FirstName, Username: user.Username,
			Category: categoryLabel, Message: request.Message,
			Screen: request.Screen, TopicSlug: request.TopicSlug,
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.telegram.SendFeedback(ctx, s.cfg.FeedbackChatID, feedback); err != nil {
				s.logger.Error("send feedback notification", "error", err)
			}
		}()
	}

	writeJSON(w, http.StatusCreated, map[string]bool{"received": true})
}
