package wal

import (
	"context"
	"runtime/debug"
	"storage/internal/concurrency"
	"storage/internal/config"
	"storage/pkg/utils"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type DataWriter interface {
	Flush(records []*Record) error
	Load(func(string) error) (*int, error)
}
type WAL struct {
	batchTimeout   time.Duration
	batchSize      int
	maxSegmentSize int
	IDRecord       atomic.Int64
	logger         *zap.Logger
	dataChan       chan *Record
	worker         *Worker
	diskManager    DataWriter
	Counter        int64
	mutex          sync.Mutex
}

func NewWAL(cfg config.WALConfig, logger *zap.Logger) (*WAL, error) {
	duration, err := time.ParseDuration(cfg.BatchTimeout)
	if err != nil {
		return nil, err
	}
	if err := utils.CheckDir(cfg.DataDirectory); err != nil {
		return nil, err
	}
	size := utils.ParseSize(cfg.MaxSegmentSize)

	worker := &Worker{
		closeCh:     make(chan struct{}),
		closeDoneCh: make(chan struct{}),
		data:        make([]*Record, 0, size),
	}

	return &WAL{
		batchTimeout:   duration,
		batchSize:      cfg.BatchSize,
		diskManager:    NewDiskManager(cfg, logger),
		maxSegmentSize: size,
		logger:         logger,
		dataChan:       make(chan *Record, cfg.BatchSize),
		worker:         worker,
	}, nil
}

type Worker struct {
	closeCh     chan struct{}
	closeDoneCh chan struct{}
	data        []*Record
}

func (w *WAL) Start(replay func(string) error) error {
	lastId, err := w.diskManager.Load(replay)
	if err != nil {
		return err
	}
	if lastId == nil {
		w.IDRecord.Store(0)
	} else {
		w.IDRecord.Store(int64(*lastId))
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
		w.flush()
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
			w.flush()

		case rec, ok := <-w.dataChan:
			if !ok {
				return
			}
			w.worker.data = append(w.worker.data, rec)
			w.Counter++

			if len(w.worker.data) >= w.batchSize {
				w.flush()
				ticker.Reset(w.batchTimeout)
			}
		}
	}
}

func (w *WAL) Stop() {
	close(w.worker.closeCh)
	<-w.worker.closeDoneCh
	close(w.dataChan)
}

func (w *WAL) Write(ctx context.Context, data string) concurrency.FutureError {
	id := w.IDRecord.Add(1)
	rec := NewRecord(data, id)

	select {
	case w.dataChan <- rec:
		return concurrency.NewFuture(rec.doneCh)

	case <-ctx.Done():
		ch := make(chan error, 1)
		ch <- ctx.Err()
		return concurrency.NewFuture(ch)
	}
}

func (w *WAL) flush() {
	if len(w.worker.data) == 0 {
		return
	}
	w.mutex.Lock()
	data := w.worker.data
	w.worker.data = w.worker.data[:0]
	w.mutex.Unlock()

	err := w.diskManager.Flush(data)
	for _, rec := range data {
		rec.doneCh <- err
	}
}
