package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (s *Server) getReminder(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	var enabled bool
	var reminderTime, timezone string
	err := s.db.QueryRow(r.Context(), `
		SELECT enabled, to_char(reminder_time, 'HH24:MI'), timezone
		FROM reminder_settings WHERE user_id = $1`, user.ID,
	).Scan(&enabled, &reminderTime, &timezone)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": false, "time": "20:00", "timezone": clientTimezone(r),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": enabled, "time": reminderTime, "timezone": timezone})
}

func (s *Server) updateReminder(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	var request struct {
		Enabled  bool   `json:"enabled"`
		Time     string `json:"time"`
		Timezone string `json:"timezone"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Проверьте настройки напоминания.")
		return
	}
	if _, err := time.Parse("15:04", request.Time); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_time", "Укажите время в формате ЧЧ:ММ.")
		return
	}
	if _, err := time.LoadLocation(request.Timezone); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_timezone", "Не удалось определить часовой пояс.")
		return
	}
	_, err := s.db.Exec(r.Context(), `
		INSERT INTO reminder_settings (user_id, enabled, reminder_time, timezone)
		VALUES ($1, $2, $3::time, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			reminder_time = EXCLUDED.reminder_time,
			timezone = EXCLUDED.timezone,
			last_sent_on = CASE WHEN reminder_settings.enabled <> EXCLUDED.enabled
				OR reminder_settings.reminder_time <> EXCLUDED.reminder_time
				THEN NULL ELSE reminder_settings.last_sent_on END,
			updated_at = now()`, user.ID, request.Enabled, request.Time, request.Timezone)
	if err != nil {
		s.logger.Error("update reminder", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить напоминание.")
		return
	}
	writeJSON(w, http.StatusOK, request)
}

func clientTimezone(r *http.Request) string {
	zone := r.Header.Get("X-Timezone")
	if _, err := time.LoadLocation(zone); err == nil {
		return zone
	}
	return "Asia/Tashkent"
}

func (s *Server) telegramWebhook(w http.ResponseWriter, r *http.Request) {
	if s.cfg.TelegramWebhookSecret != "" && r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != s.cfg.TelegramWebhookSecret {
		writeError(w, http.StatusUnauthorized, "invalid_webhook_secret", "Unauthorized")
		return
	}
	var update struct {
		Message *struct {
			Text string `json:"text"`
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_update", "Invalid update")
		return
	}
	if update.Message != nil && strings.HasPrefix(update.Message.Text, "/start") && s.telegram.Enabled() {
		if err := s.telegram.SendStart(r.Context(), update.Message.Chat.ID); err != nil {
			s.logger.Error("send Telegram start message", "error", err)
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
