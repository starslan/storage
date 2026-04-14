package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"storage/internal/app"
	"storage/internal/config"
	"storage/internal/logger"

	"go.uber.org/zap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		logger.Fatal("Failed to start application", zap.Error(err))
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
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

	if err := scanner.Err(); err != nil {
		logger.Error("Error I/O scanner", zap.Error(err))
	}
	logger.Info("Application finished")
}
