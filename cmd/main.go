package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"storage/internal/config"
	"storage/internal/database"
	"storage/internal/database/compute"
	"storage/internal/database/compute/parser"
	"storage/internal/database/storage"
	"storage/internal/database/storage/engine/memory"

	"go.uber.org/zap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Application started")

	psr := parser.NewParser(logger, 300)
	cmt, err := compute.NewCompute(psr)
	if err != nil {
		logger.Fatal("Failed to create compute", zap.Error(err))
	}
	engine, err := memory.NewEngine(&cfg.Engine)
	if err != nil {
		logger.Fatal("Failed to create engine", zap.Error(err))
	}
	str, err := storage.NewStorage(engine, logger)
	if err != nil {
		logger.Fatal("Failed to create storage", zap.Error(err))
	}
	DB, err := database.NewDB(logger, cmt, str)
	if err != nil {
		logger.Fatal("Failed to create database", zap.Error(err))
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(" DB> ")
		if !scanner.Scan() {
			break
		}
		request := scanner.Text()

		logger.Debug("Request:", zap.String("request", request))

		result := DB.HandleQuery(ctx, request)

		logger.Debug("Result:", zap.String("result", result))
		fmt.Println(result)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error I/O scanner", zap.Error(err))
	}
	logger.Info("Application finished")
}
