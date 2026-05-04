package wal

import (
	"context"
	"fmt"
	"runtime/debug"
	"storage/internal/config"
	"storage/pkg/utils"
	"strconv"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type DataWriter interface {
	Flush(records []record) error
	Load(func(string) error) error
}
type WAL struct {
	batchTimeout   time.Duration
	maxSegmentSize int
	logger         *zap.Logger
	dataChan       chan record
	worker         *Worker
	diskManager    DataWriter
	Counter        atomic.Int64
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
		batchTimeout:   duration,
		diskManager:    NewDiskManager(cfg, logger, size),
		maxSegmentSize: size,
		logger:         logger,
		dataChan:       make(chan record, cfg.BatchSize/3),
		worker:         worker,
	}, nil
}

type record struct {
	id     int64
	data   string
	doneCh chan error
}

func (record *record) String() string {
	return strconv.FormatInt(record.id, 10) + "_" + record.data
}

type Worker struct {
	closeCh     chan struct{}
	closeDoneCh chan struct{}
	data        []record
}

func (w *WAL) Start(replay func(string) error) error {
	err := w.diskManager.Load(replay)
	if err != nil {
		return err
	}
	go w.startWorker()
	return nil
}

func (w *WAL) startWorker() {
	ticker := time.NewTicker(w.batchTimeout)
	defer func() {
		needRestart := false
		if r := recover(); r != nil {
			w.logger.Warn("recovered from panic", zap.String("stack", string(debug.Stack())))
			needRestart = true
		}
		ticker.Stop()
		err := w.flush()
		if err != nil {
			w.logger.Error("failed to flush WAL", zap.Error(err))
		}
		if !needRestart {
			close(w.worker.closeDoneCh)
			return
		}

		go w.startWorker()

	}()

	for {
		select {
		case <-w.worker.closeCh:
			w.logger.Info("shutting down WAL")
			return
		case <-ticker.C:
			err := w.flush()
			if err != nil {
				w.logger.Error("failed to flush WAL", zap.Error(err))
			}

		case rec, ok := <-w.dataChan:
			if !ok {
				return
			}
			rec.id = w.Counter.Load()
			w.worker.data = append(w.worker.data, rec)
			w.Counter.Add(1)
			if len(w.worker.data) >= w.maxSegmentSize {
				err := w.flush()
				if err != nil {
					w.logger.Error("failed to flush WAL", zap.Error(err))
				}
			}
		}
	}
}

func (w *WAL) Stop() {
	close(w.worker.closeCh)
	<-w.worker.closeDoneCh
	close(w.dataChan)
}

func (w *WAL) Write(ctx context.Context, data string) error {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	rec := record{data: data, doneCh: make(chan error, 1)}
	select {
	case <-w.worker.closeCh:
		return fmt.Errorf("wal is stopped")
	case w.dataChan <- rec:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-rec.doneCh:
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *WAL) flush() error {
	data := w.worker.data

	if len(data) == 0 {
		return nil
	}

	err := w.diskManager.Flush(data)
	for _, rec := range data {
		rec.doneCh <- err
	}
	w.worker.data = w.worker.data[:0]
	return err
}
