package database

import (
	"context"
	"errors"
	"fmt"
	"storage/internal/database/compute"
	"storage/internal/database/storage"
	"time"

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

type WAL interface {
	Write(context.Context, string) error
	Start(func(string) error) error
	Stop()
}

type DB struct {
	logger  *zap.Logger
	compute Compute
	storage Storage
	wal     WAL
}

func NewDB(logger *zap.Logger, parser Compute, storage Storage, wal WAL) (*DB, error) {
	return &DB{
		logger:  logger,
		compute: parser,
		storage: storage,
		wal:     wal,
	}, nil
}

func (d *DB) HandleQuery(ctx context.Context, queryStr string) string {
	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	query, err := d.compute.Parse(queryStr)
	if err != nil {
		d.logger.Error("Failed to parse query", zap.Error(err))
		return fmt.Sprintf("Failed to parse query: %s", queryStr)
	}
	switch query.CommandID() {
	case compute.SetCommandID:
		if err := d.writeToWAL(ctxTimeout, queryStr); err != nil {
			return fmt.Sprintf("Failed to write query: %s", queryStr)
		}
		return d.handleSetQuery(ctxTimeout, query)
	case compute.GetCommandID:
		return d.handleGetQuery(ctxTimeout, query)
	case compute.DelCommandID:
		if err := d.writeToWAL(ctxTimeout, queryStr); err != nil {
			return fmt.Sprintf("Failed to write query: %s", queryStr)
		}
		return d.handleDelQuery(ctxTimeout, query)
	default:
	}

	d.logger.Error(
		"compute layer is incorrect",
		zap.Int("command_id", query.CommandID()),
	)
	return "[error] internal error"
}

func (d *DB) HandleWalQuery(queryStr string) string {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query, err := d.compute.Parse(queryStr)
	if err != nil {
		d.logger.Error("Failed to parse query", zap.Error(err))
		return fmt.Sprintf("Failed to parse query: %s", queryStr)
	}
	switch query.CommandID() {
	case compute.SetCommandID:
		return d.handleSetQuery(ctxTimeout, query)
	case compute.GetCommandID:
		return d.handleGetQuery(ctxTimeout, query)
	case compute.DelCommandID:
		return d.handleDelQuery(ctxTimeout, query)
	default:
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

func (d *DB) Start(_ context.Context) error {
	if d.wal != nil {
		err := d.wal.Start(func(q string) error {
			d.HandleWalQuery(q)
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func (d *DB) Stop() error {
	if d.wal != nil {
		d.wal.Stop()
	}
	return nil
}

func (d *DB) writeToWAL(ctx context.Context, queryStr string) error {
	if d.wal == nil {
		return nil
	}

	if err := d.wal.Write(ctx, queryStr); err != nil {
		d.logger.Error("Failed to write query to wal", zap.Error(err), zap.String("query", queryStr))
		return err
	}
	return nil
}
