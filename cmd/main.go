package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
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
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Application started")

	psr := parser.NewParser(logger, 300)
	cmt, err := compute.NewCompute(logger, psr)
	if err != nil {
		logger.Error("Failed to create compute", zap.Error(err))
	}
	eng, err := memory.NewEngine(logger)
	if err != nil {
		logger.Error("Failed to create engine", zap.Error(err))
	}
	str, err := storage.NewStorage(eng, logger)
	if err != nil {
		logger.Error("Failed to create storage", zap.Error(err))
	}
	DB, err := database.NewDB(logger, cmt, str)
	fmt.Println(DB)
	if err != nil {
		logger.Error("Failed to create database", zap.Error(err))
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
}
