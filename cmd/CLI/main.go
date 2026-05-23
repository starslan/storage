package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"storage/internal/app"
	"storage/internal/config"
	"storage/internal/logger"
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
	config.ApplyArguments(cfg)

	fmt.Println(cfg)

	logger, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	defer logger.Sync()

	logger.Info("Application started")
	logger.Info("Config ", zap.Any("cfg", cfg))

	app, err := app.NewApp(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create application", zap.Error(err))
	}

	err = app.Start(ctx)
	if err != nil {
		logger.Fatal("Failed to start application", zap.Error(err))
	}

	scanner := bufio.NewScanner(os.Stdin)
	running := true
	for {
		select {
		case <-interruptCh:
			logger.Info("Interrupt received")
			running = false
		default:
		}
		if !running {
			fmt.Println("Exiting...")
			break
		}
		fmt.Print(" DB> ")
		if !scanner.Scan() {
			break
		}
		request := scanner.Text()

		logger.Debug("Request:", zap.String("request", request))

		result := app.DB.HandleQuery(ctx, request)

		logger.Debug("Result:", zap.String("result", result))
		fmt.Println(result)
	}

	err = app.Stop()
	if err != nil {
		logger.Error("Failed to stop application", zap.Error(err))
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error I/O scanner", zap.Error(err))
	}
	logger.Info("Application finished")
}
