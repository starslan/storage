package wal

import (
	"fmt"
	"runtime/debug"
	"storage/internal/config"
	"storage/pkg/utils"
	"time"

	"go.uber.org/zap"
)

type DataWriter interface {
	Flush(records []record) error
}
type WAL struct {
	batchTimeout   time.Duration
	maxSegmentSize int
	logger         *zap.Logger
	dataChan       chan record
	worker         *Worker
	diskManager    DataWriter
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
		batchTimeout:   duration,
		diskManager:    NewDiskManager(cfg),
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
type Worker struct {
	closeCh     chan struct{}
	closeDoneCh chan struct{}
	data        []record
}

func (w *WAL) Start() {
	go func() {
		ticker := time.NewTicker(w.batchTimeout)
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
				rec.id = w.Counter
				w.worker.data = append(w.worker.data, rec)
				w.Counter++
				if len(w.worker.data) >= w.maxSegmentSize {
					err := w.flush()
					if err != nil {
						w.logger.Error("failed to flush WAL", zap.Error(err))
					}
				}
			}
		}
	}()
}

func (w *WAL) Stop() {
	close(w.worker.closeCh)
	<-w.worker.closeDoneCh
	close(w.dataChan)
}

func (w *WAL) Write(data string) error {
	timer := time.NewTimer(3000 * time.Millisecond)
	defer timer.Stop()

	rec := record{data: data, doneCh: make(chan error, 1)}
	select {
	case <-w.worker.closeCh:
		return fmt.Errorf("wal is stopped")
	case w.dataChan <- rec:
	}

	select {
	case <-timer.C:
		return fmt.Errorf("timed out write")
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
