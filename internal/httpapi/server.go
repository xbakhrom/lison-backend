package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xbakhrom/lison-backend/internal/auth"
	"github.com/xbakhrom/lison-backend/internal/config"
	"github.com/xbakhrom/lison-backend/internal/telegram"
)

type contextKey string

const userKey contextKey = "telegram-user"

type Server struct {
	cfg      config.Config
	db       *pgxpool.Pool
	telegram *telegram.Client
	logger   *slog.Logger
}

func New(cfg config.Config, db *pgxpool.Pool, telegramClient *telegram.Client, logger *slog.Logger) *Server {
	return &Server{cfg: cfg, db: db, telegram: telegramClient, logger: logger}
}

func (s *Server) Routes() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(s.cors)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Post("/api/v1/telegram/webhook", s.telegramWebhook)

	router.Route("/api/v1", func(api chi.Router) {
		api.Use(s.authenticate)
		api.Post("/feedback", s.submitFeedback)
		api.Get("/topics", s.listTopics)
		api.Get("/topics/{slug}", s.getTopic)
		api.Put("/topics/{topicID}/answers/{fieldID}", s.saveAnswer)
		api.Put("/topics/{topicID}/checklist/{itemID}", s.saveChecklist)
		api.Get("/grammar", s.listGrammarTopics)
		api.Get("/grammar/{slug}", s.getGrammarTopic)
		api.Post("/grammar/{topicID}/games", s.finishGrammarGame)
		api.Get("/cards", s.listCards)
		api.Post("/cards/bulk", s.addCards)
		api.Delete("/cards/{cardID}", s.deleteCard)
		api.Get("/reviews/due", s.listDueCards)
		api.Post("/reviews/{cardID}", s.reviewCard)
		api.Get("/reminder", s.getReminder)
		api.Put("/reminder", s.updateReminder)
	})
	return router
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user auth.User
		authorization := r.Header.Get("Authorization")
		if strings.HasPrefix(authorization, "tma ") {
			var err error
			user, err = auth.ValidateInitData(strings.TrimPrefix(authorization, "tma "), s.cfg.TelegramBotToken, s.cfg.TelegramAuthMaxAge, time.Now())
			if err != nil {
				writeError(w, http.StatusUnauthorized, "telegram_auth_failed", "Не удалось подтвердить данные Telegram.")
				return
			}
		} else if s.cfg.Environment == "development" {
			user = auth.User{ID: s.cfg.DevUserID, FirstName: "Demo"}
		} else {
			writeError(w, http.StatusUnauthorized, "authentication_required", "Откройте приложение через Telegram.")
			return
		}

		_, err := s.db.Exec(r.Context(), `
			INSERT INTO users (telegram_id, first_name, username, language_code)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (telegram_id) DO UPDATE SET
				first_name = EXCLUDED.first_name,
				username = EXCLUDED.username,
				language_code = EXCLUDED.language_code,
				updated_at = now()`,
			user.ID, user.FirstName, user.Username, user.LanguageCode,
		)
		if err != nil {
			s.logger.Error("upsert user", "error", err)
			writeError(w, http.StatusInternalServerError, "database_error", "Не удалось открыть профиль.")
			return
		}

		ctx := context.WithValue(r.Context(), userKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func currentUser(ctx context.Context) auth.User {
	user, _ := ctx.Value(userKey).(auth.User)
	return user
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == s.cfg.FrontendOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Timezone")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) == nil {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
