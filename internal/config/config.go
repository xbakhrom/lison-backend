package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment           string
	Port                  string
	DatabaseURL           string
	FrontendOrigin        string
	TelegramBotToken      string
	TelegramWebhookSecret string
	FeedbackChatID        int64
	MiniAppURL            string
	DevUserID             int64
	TelegramAuthMaxAge    time.Duration
}

func Load() Config {
	return Config{
		Environment:           env("APP_ENV", "development"),
		Port:                  env("PORT", "8080"),
		DatabaseURL:           env("DATABASE_URL", "postgres://lison:lison@localhost:55432/lison?sslmode=disable"),
		FrontendOrigin:        env("FRONTEND_ORIGIN", "http://localhost:5173"),
		TelegramBotToken:      os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		FeedbackChatID:        envInt64("FEEDBACK_CHAT_ID", 0),
		MiniAppURL:            env("MINI_APP_URL", "http://localhost:5173"),
		DevUserID:             envInt64("DEV_USER_ID", 1001),
		TelegramAuthMaxAge:    24 * time.Hour,
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(os.Getenv(key), 10, 64)
	if err != nil {
		return fallback
	}
	return value
}
