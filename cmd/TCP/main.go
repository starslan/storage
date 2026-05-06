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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, syscall.SIGINT, syscall.SIGTERM)
	defer close(interruptCh)

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

	err = app.Start(ctx)
	if err != nil {
		logger.Fatal("Failed to start application", zap.Error(err))
	}

	srv, err := server.NewServer(app, cfg.Network)
	if err != nil {
		logger.Fatal("Failed to create new server", zap.Error(err))
	}

	go func() {
		if err := srv.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	fmt.Println("Server started")

	<-interruptCh
	fmt.Println("Signal interrupt")

	cancel()
	err = app.Stop()
	if err != nil {
		logger.Fatal("Failed to stop app", zap.Error(err))
	}
	err = srv.Stop()
	if err != nil {
		logger.Fatal("Failed to stop server", zap.Error(err))
	}

}
