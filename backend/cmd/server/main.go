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

	"studytrack/backend/internal/auth"
	"studytrack/backend/internal/config"
	"studytrack/backend/internal/database"
	"studytrack/backend/internal/httpapi"
	"studytrack/backend/internal/store"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	dataStore := store.New(pool)
	authService := auth.NewService([]byte(cfg.JWTSecret), 24*time.Hour)
	api := httpapi.New(dataStore, authService, logger, cfg.CORSOrigins)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("studytrack API started", "address", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
