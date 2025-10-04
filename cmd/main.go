package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"qonto/config"
	"qonto/internal/core"
	"qonto/internal/http"
	"qonto/internal/sqlite"
)

func main() {
	ctx := context.Background()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	cfg, err := config.Load()
	if err != nil {
		slog.ErrorContext(ctx, "failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.Level(cfg.LogLevel),
	}))
	slog.SetDefault(logger)

	logger.InfoContext(ctx, "Starting application")

	dbClient, err := sqlite.NewClient(cfg.Database)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create db client", "error", err)
		os.Exit(1)
	}

	accountRepository := sqlite.NewAccountStore(dbClient.DB())
	service := core.NewService(accountRepository)
	httpServer := http.NewServer(service, logger, cfg.HTTP)

	if err = httpServer.Start(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to start http server", "error", err)
		os.Exit(1)
	}

	<-stop

	logger.InfoContext(ctx, "Shutting down...")

	if err = httpServer.Stop(ctx); err != nil {
		logger.ErrorContext(ctx, "Error stopping HTTP server", "error", err)
	}

	logger.InfoContext(ctx, "Application shutdown complete")
}
