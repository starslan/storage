package app

import (
	"storage/internal/config"
	"storage/internal/database"
	"storage/internal/database/compute"
	"storage/internal/database/compute/parser"
	"storage/internal/database/storage"
	"storage/internal/database/storage/engine/memory"
	"storage/internal/wal"

	"go.uber.org/zap"
)

type App struct {
	DB     *database.DB
	Logger *zap.Logger
}

func NewApp(cfg *config.Config, logger *zap.Logger) (*App, error) {

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

	var walLog *wal.WAL
	if cfg.WALConfig.Enable {
		walLog, err = wal.NewWAL(cfg.WALConfig, logger)
		if err != nil {
			return nil, err
		}
	}

	db, err := database.NewDB(logger, cmt, str, walLog)
	if err != nil {
		return nil, err
	}

	return &App{
		DB:     db,
		Logger: logger,
	}, nil
}

func (app *App) Start() error {
	return app.DB.Start()
}

func (app *App) Stop() error {
	return app.DB.Stop()
}
