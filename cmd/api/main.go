package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xbakhrom/lison-backend/internal/config"
	"github.com/xbakhrom/lison-backend/internal/database"
	"github.com/xbakhrom/lison-backend/internal/httpapi"
	"github.com/xbakhrom/lison-backend/internal/reminder"
	"github.com/xbakhrom/lison-backend/internal/telegram"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		logger.Error("migrate database", "error", err)
		os.Exit(1)
	}

	telegramClient := telegram.NewClient(cfg.TelegramBotToken, cfg.MiniAppURL)
	api := httpapi.New(cfg, db, telegramClient, logger)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go reminder.New(db, telegramClient, logger).Run(ctx)
	go func() {
		logger.Info("API listening", "port", cfg.Port, "environment", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve API", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown API", "error", err)
	}
}
