package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/xbakhrom/lison-backend/internal/assistant"
	"github.com/xbakhrom/lison-backend/internal/gemini"
)

const (
	// The browser must open the WebSocket almost immediately after asking for a
	// token; a long window would make a leaked token useful to someone else.
	assistantNewSessionWindow = time.Minute
	// A single Live connection lives ~10 minutes, after which the client asks
	// for a fresh token and resumes. The small margin covers a slow reconnect
	// without handing out materially more audio time than one connection needs.
	assistantTokenLifetime = 11 * time.Minute
	// Prompt size guard: a whole lesson's vocabulary, but never more.
	assistantMaxTopicWords = 60
	// Largest duration a single client report is allowed to claim.
	assistantMaxReportedSeconds = 30 * 60
)

// assistantMaxDailyTokens caps how many tokens one learner can mint per day.
//
// Spent minutes are reported by the client and are therefore spoofable, so this
// is what actually bounds the spend: a client that never reports usage still
// cannot mint more than this many tokens, and each one is only good for one
// connection of assistantTokenLifetime. The allowance is the number of
// connections an honest conversation needs (one per ten minutes of quota) plus
// headroom for reconnects on a flaky mobile network.
func assistantMaxDailyTokens(dailyMinutes int) int {
	return max(dailyMinutes/10+3, 4)
}

type assistantTokenResponse struct {
	Token               string `json:"token"`
	Model               string `json:"model"`
	ExpiresAt           string `json:"expiresAt"`
	NewSessionExpiresAt string `json:"newSessionExpiresAt"`
	RemainingSeconds    int    `json:"remainingSeconds"`
	DailyLimitSeconds   int    `json:"dailyLimitSeconds"`
}

func (s *Server) createAssistantToken(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	if !s.gemini.Configured() {
		writeError(w, http.StatusServiceUnavailable, "assistant_disabled", "Голосовой помощник сейчас недоступен.")
		return
	}

	var request struct {
		TopicSlug string `json:"topicSlug"`
	}
	if r.ContentLength > 0 {
		if err := decodeJSON(w, r, &request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Некорректный запрос.")
			return
		}
	}

	limit := s.cfg.AssistantDailyMinutes * 60
	used, tokensMinted, err := s.assistantUsage(r.Context(), user.ID, localToday(r))
	if err != nil {
		s.logger.Error("assistant usage", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось открыть разговор.")
		return
	}
	remaining := limit - used
	if remaining <= 0 || tokensMinted >= assistantMaxDailyTokens(s.cfg.AssistantDailyMinutes) {
		writeError(w, http.StatusTooManyRequests, "assistant_quota_exceeded", "Дневной лимит разговоров исчерпан. Возвращайтесь завтра.")
		return
	}

	sessionContext, err := s.assistantContext(r, user.ID, request.TopicSlug)
	if err != nil {
		s.logger.Error("assistant context", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось открыть разговор.")
		return
	}

	// No language is pinned even though Maks speaks Russian: he still has to be
	// able to answer in Uzbek on the rare occasions the learner asks for it, and
	// a pinned languageCode would mangle those few sentences. The persona is
	// what keeps him in Russian.
	token, err := s.gemini.CreateToken(r.Context(), gemini.SessionConstraints{
		SystemInstruction: assistant.BuildSystemInstruction(sessionContext),
		VoiceName:         "Puck",
	}, assistantNewSessionWindow, assistantTokenLifetime)
	if err != nil {
		if errors.Is(err, gemini.ErrNotConfigured) {
			writeError(w, http.StatusServiceUnavailable, "assistant_disabled", "Голосовой помощник сейчас недоступен.")
			return
		}
		s.logger.Error("mint assistant token", "error", err)
		writeError(w, http.StatusBadGateway, "assistant_unavailable", "Не удалось связаться с помощником. Попробуйте ещё раз.")
		return
	}

	// Recorded after the mint succeeds, so a Google outage does not eat the
	// learner's allowance.
	if err := s.assistantCountToken(r.Context(), user.ID, localToday(r)); err != nil {
		s.logger.Error("assistant count token", "error", err)
	}

	writeJSON(w, http.StatusOK, assistantTokenResponse{
		Token:               token.Name,
		Model:               token.Model,
		ExpiresAt:           token.ExpiresAt.Format(time.RFC3339),
		NewSessionExpiresAt: token.NewSessionExpiresAt.Format(time.RFC3339),
		RemainingSeconds:    remaining,
		DailyLimitSeconds:   limit,
	})
}

func (s *Server) reportAssistantSession(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	var request struct {
		Seconds int `json:"seconds"`
	}
	if err := decodeJSON(w, r, &request); err != nil || request.Seconds < 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Некорректная длительность разговора.")
		return
	}
	if request.Seconds > assistantMaxReportedSeconds {
		request.Seconds = assistantMaxReportedSeconds
	}

	today := localToday(r)
	_, err := s.db.Exec(r.Context(), `
		INSERT INTO user_assistant_usage (user_id, usage_date, seconds, sessions)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT (user_id, usage_date) DO UPDATE
		SET seconds = user_assistant_usage.seconds + EXCLUDED.seconds,
		    updated_at = now()`, user.ID, today, request.Seconds)
	if err != nil {
		s.logger.Error("assistant report session", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить разговор.")
		return
	}

	used, _, err := s.assistantUsage(r.Context(), user.ID, today)
	if err != nil {
		s.logger.Error("assistant usage", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить разговор.")
		return
	}
	limit := s.cfg.AssistantDailyMinutes * 60
	writeJSON(w, http.StatusOK, map[string]int{
		"usedSeconds":       used,
		"remainingSeconds":  max(limit-used, 0),
		"dailyLimitSeconds": limit,
	})
}

// assistantUsage returns the seconds spent and the number of tokens minted by
// this learner today.
// assistantLevels are the only levels Maks may record, matching the levels the
// authored content is labelled with.
var assistantLevels = map[string]bool{"A1": true, "A2": true, "B1": true, "B2": true}

// setAssistantLevel lets Maks record what he heard during a placement chat, so
// later sessions start at the right difficulty.
func (s *Server) setAssistantLevel(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	var request struct {
		Level string `json:"level"`
	}
	if err := decodeJSON(w, r, &request); err != nil || !assistantLevels[strings.ToUpper(request.Level)] {
		writeError(w, http.StatusBadRequest, "invalid_request", "Неизвестный уровень.")
		return
	}
	level := strings.ToUpper(request.Level)
	if _, err := s.db.Exec(r.Context(),
		"UPDATE users SET russian_level = $2, updated_at = now() WHERE telegram_id = $1", user.ID, level); err != nil {
		s.logger.Error("set assistant level", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось сохранить уровень.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"level": level})
}

func (s *Server) assistantUsage(ctx context.Context, userID int64, day time.Time) (seconds, tokens int, err error) {
	err = s.db.QueryRow(ctx, `
		SELECT seconds, sessions FROM user_assistant_usage
		WHERE user_id = $1 AND usage_date = $2`, userID, day).Scan(&seconds, &tokens)
	if isNoRows(err) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	return seconds, tokens, nil
}

func (s *Server) assistantCountToken(ctx context.Context, userID int64, day time.Time) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO user_assistant_usage (user_id, usage_date, seconds, sessions)
		VALUES ($1, $2, 0, 1)
		ON CONFLICT (user_id, usage_date) DO UPDATE
		SET sessions = user_assistant_usage.sessions + 1, updated_at = now()`, userID, day)
	return err
}

func (s *Server) assistantContext(r *http.Request, userID int64, topicSlug string) (assistant.Context, error) {
	ctx := r.Context()
	result := assistant.Context{}

	if err := s.db.QueryRow(ctx, `
		SELECT first_name, russian_level FROM users WHERE telegram_id = $1`, userID,
	).Scan(&result.FirstName, &result.Level); err != nil && !isNoRows(err) {
		return result, err
	}

	if err := s.db.QueryRow(ctx, `
		SELECT count(*)::int FROM user_cards WHERE user_id = $1 AND due_date <= $2`,
		userID, localToday(r),
	).Scan(&result.DueCards); err != nil {
		return result, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT t.title
		FROM user_cards c
		JOIN vocabulary_items v ON v.id = c.vocabulary_item_id
		JOIN topics t ON t.id = v.topic_id
		WHERE c.user_id = $1
		GROUP BY t.id, t.title
		ORDER BY max(c.created_at) DESC
		LIMIT 5`, userID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			return result, err
		}
		result.RecentTopics = append(result.RecentTopics, title)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}

	if topicSlug == "" {
		return result, nil
	}

	topic := assistant.TopicContext{Slug: topicSlug}
	if err := s.db.QueryRow(ctx, `
		SELECT id, title, summary FROM topics
		WHERE slug = $1 AND status = 'published'`, topicSlug,
	).Scan(&topic.ID, &topic.Title, &topic.Summary); err != nil {
		// An unknown slug is not worth failing the whole session over.
		if isNoRows(err) {
			return result, nil
		}
		return result, err
	}

	wordRows, err := s.db.Query(ctx, `
		SELECT v.russian, v.uzbek, uc.id IS NOT NULL
		FROM vocabulary_items v
		LEFT JOIN user_cards uc ON uc.vocabulary_item_id = v.id AND uc.user_id = $2
		WHERE v.topic_id = $1 AND v.active
		ORDER BY v.position
		LIMIT $3`, topic.ID, userID, assistantMaxTopicWords)
	if err != nil {
		return result, err
	}
	defer wordRows.Close()
	for wordRows.Next() {
		var word assistant.Word
		var added bool
		if err := wordRows.Scan(&word.Russian, &word.Uzbek, &added); err != nil {
			return result, err
		}
		topic.Words = append(topic.Words, word)
		if !added {
			topic.Unlearned = append(topic.Unlearned, word.Russian)
		}
	}
	if err := wordRows.Err(); err != nil {
		return result, err
	}

	result.Topic = &topic
	return result, nil
}
