package database

import (
	"context"
	"errors"
	"fmt"
	"storage/internal/database/compute"
	"storage/internal/database/storage"

	"go.uber.org/zap"
)

type Compute interface {
	Parse(string) (compute.Query, error)
}

type Storage interface {
	Set(context.Context, string, string) error
	Get(context.Context, string) (string, error)
	Del(context.Context, string) error
}

type DB struct {
	logger  *zap.Logger
	parser  Compute
	storage Storage
}

func NewDB(logger *zap.Logger, parser Compute, storage Storage) (*DB, error) {
	return &DB{
		logger:  logger,
		parser:  parser,
		storage: storage,
	}, nil
}

func (d *DB) HandleQuery(ctx context.Context, queryStr string) string {
	query, err := d.parser.Parse(queryStr)
	if err != nil {
		d.logger.Error("Failed to parse query", zap.Error(err))
		return fmt.Sprintf("Failed to parse query: %s", queryStr)
	}
	switch query.CommandID() {
	case compute.SetCommandID:
		return d.handleSetQuery(ctx, query)
	case compute.GetCommandID:
		return d.handleGetQuery(ctx, query)
	case compute.DelCommandID:
		return d.handleDelQuery(ctx, query)
	}

	d.logger.Error(
		"compute layer is incorrect",
		zap.Int("command_id", query.CommandID()),
	)
	return "[error] internal error"
}

func (d *DB) handleSetQuery(ctx context.Context, query compute.Query) string {
	arguments := query.Arguments()
	if err := d.storage.Set(ctx, arguments[0], arguments[1]); err != nil {
		d.logger.Error("Failed to set query", zap.Error(err))
		return fmt.Sprintf("[error] %s", err)
	}

	return "[ok]"
}

func (d *DB) handleGetQuery(ctx context.Context, query compute.Query) string {
	arguments := query.Arguments()
	value, err := d.storage.Get(ctx, arguments[0])
	if errors.Is(err, storage.ErrorNotFound) {
		return "[not found]"
	} else if err != nil {
		d.logger.Error("Failed to get query", zap.Error(err))
		return fmt.Sprintf("[error] %s", err.Error())
	}

	return fmt.Sprintf("[ok] %s", value)
}

func (d *DB) handleDelQuery(ctx context.Context, query compute.Query) string {
	arguments := query.Arguments()
	if err := d.storage.Del(ctx, arguments[0]); err != nil {
		d.logger.Error("Failed to delete query", zap.Error(err))
		return fmt.Sprintf("[error] %s", err.Error())
	}

	return "[ok]"
}
