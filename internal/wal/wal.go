package wal

import (
	"context"
	"storage/internal/config"
	"storage/pkg/utils"
	"time"

	"go.uber.org/zap"
)

type WAL struct {
	BatchTimeout   time.Duration
	MaxSegmentSize int
	DataPath       string
	logger         *zap.Logger
	data           []string
	dataChan       chan string
}

func NewWAL(cfg config.WALConfig, logger *zap.Logger) (*WAL, error) {
	duration, err := time.ParseDuration(cfg.BatchTimeout)
	if err != nil {
		return nil, err
	}
	err = utils.CheckDir(cfg.DataDirectory)
	if err != nil {
		return nil, err
	}
	size := utils.ParseSize(cfg.MaxSegmentSize)

	return &WAL{
		BatchTimeout:   duration,
		DataPath:       cfg.DataDirectory,
		MaxSegmentSize: size,
		logger:         logger,
		dataChan:       make(chan string, cfg.BatchSize/3),
		data:           make([]string, 0, cfg.BatchSize),
	}, nil
}

func (W WAL) Start(_ context.Context) error {
	go func() {
		var _ *time.Timer
		for range W.dataChan {
			_ = <-W.dataChan
			if len(W.data) == 0 {
				_ = time.NewTimer(W.BatchTimeout)
			}

		}
	}()

	return nil
}

func (W WAL) Stop() {
	close(W.dataChan)
}

func (W WAL) Write(ctx context.Context, data string) error {
	return nil
}
