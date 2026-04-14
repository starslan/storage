package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"storage/internal/app"
	"storage/internal/config"
	"storage/internal/logger"
	"storage/internal/server"
	"syscall"

	"go.uber.org/zap"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	logger, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	defer func() {
		_ = logger.Sync()
	}()

	app, err := app.NewApp(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to start application", zap.Error(err))
	}

	srv, err := server.NewServer(app, cfg.Network)
	if err != nil {
		logger.Fatal("Failed to create new server", zap.Error(err))
	}

	if err := srv.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal("Failed to start server", zap.Error(err))
	}

	_ = srv.Stop()
}
