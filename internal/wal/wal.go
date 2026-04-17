package wal

import (
	"context"
	"fmt"
	"runtime/debug"
	"storage/internal/config"
	"storage/pkg/utils"
	"time"

	"go.uber.org/zap"
)

type WAL struct {
	BatchTimeout   time.Duration
	MaxSegmentSize int
	logger         *zap.Logger
	dataChan       chan string
	worker         *Worker
	diskManager    *DiskManager
	Counter        int64
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

	worker := &Worker{
		closeCh:     make(chan struct{}),
		closeDoneCh: make(chan struct{}),
		data:        make([]record, 0, size),
	}

	return &WAL{
		BatchTimeout:   duration,
		diskManager:    NewDiskManager(cfg),
		MaxSegmentSize: size,
		logger:         logger,
		dataChan:       make(chan string, cfg.BatchSize/3),
		worker:         worker,
	}, nil
}

type record struct {
	id   int64
	data string
}
type Worker struct {
	closeCh     chan struct{}
	closeDoneCh chan struct{}
	data        []record
}

func (w *WAL) Start(_ context.Context) *Worker {

	go func() {
		ticker := time.NewTicker(w.BatchTimeout)
		defer func() {
			if r := recover(); r != nil {
				w.logger.Warn("recovered from panic", zap.String("stack", string(debug.Stack())))
			}
			ticker.Stop()
			err := w.flush()
			if err != nil {
				w.logger.Error("failed to flush WAL", zap.Error(err))
			}
			close(w.worker.closeDoneCh)
		}()

		for {
			select {
			case <-w.worker.closeCh:
				return
			case <-ticker.C:
				_ = w.flush()

			case str := <-w.dataChan:
				w.worker.data = append(w.worker.data, record{
					id:   w.Counter,
					data: str,
				})
				w.Counter++
				if len(w.worker.data) >= w.MaxSegmentSize {
					_ = w.flush()
				}
			}
		}
	}()

	return w.worker
}

func (w *WAL) Stop() {
	close(w.worker.closeCh)
	<-w.worker.closeDoneCh
}

func (w *WAL) Write(data string) error {
	select {
	case <-w.worker.closeCh:
		return fmt.Errorf("wal is stopped")
	case w.dataChan <- data:
		return nil
	}
}

func (w *WAL) flush() error {
	return w.diskManager.Flush(w.worker.data)
}
