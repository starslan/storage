package app

import (
	"storage/internal/config"
	"storage/internal/database"
	"storage/internal/database/compute"
	"storage/internal/database/compute/parser"
	"storage/internal/database/storage"
	"storage/internal/database/storage/engine/memory"
	log "storage/internal/logger"

	"go.uber.org/zap"
)

type App struct {
	DB     *database.DB
	Logger *zap.Logger
}

func NewApp(cfg *config.Config) (*App, error) {

	logger, err := log.NewLogger(cfg.Logger)
	if err != nil {
		return nil, err
	}

	psr := parser.NewParser(logger, cfg.Parser)
	cmt, err := compute.NewCompute(psr)
	if err != nil {
		return nil, err
	}

	engine, err := memory.NewEngine(&cfg.Engine)
	if err != nil {
		return nil, err
	}

	str, err := storage.NewStorage(engine, logger)
	if err != nil {
		return nil, err
	}

	db, err := database.NewDB(logger, cmt, str)
	if err != nil {
		return nil, err
	}

	return &App{
		DB:     db,
		Logger: logger,
	}, nil
}
