package wal

import (
	"context"
	"fmt"
	"runtime/debug"
	"storage/internal/config"
	"storage/pkg/utils"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type DataWriter interface {
	Flush(records []record) error
	Load(func(string) error) (*int, error)
}
type WAL struct {
	batchTimeout   time.Duration
	batchSize      int
	maxSegmentSize int
	IDRecord       atomic.Int64
	logger         *zap.Logger
	dataChan       chan record
	worker         *Worker
	diskManager    DataWriter
	Counter        int64
	mutex          sync.Mutex
	flushCh        chan struct{}
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
		data:        make([]record, 0, size),
	}

	return &WAL{
		batchTimeout:   duration,
		batchSize:      cfg.BatchSize,
		diskManager:    NewDiskManager(cfg, logger),
		maxSegmentSize: size,
		logger:         logger,
		dataChan:       make(chan record, cfg.BatchSize),
		worker:         worker,
		flushCh:        make(chan struct{}, 1),
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
		if err := w.flush(); err != nil {
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
			w.triggerFlush()

		case rec, ok := <-w.dataChan:
			if !ok {
				return
			}

			rec.id = w.IDRecord.Add(1)
			w.worker.data = append(w.worker.data, rec)
			w.Counter++

			if len(w.worker.data) >= w.batchSize {
				w.triggerFlush()
			}
		case <-w.flushCh:
			if err := w.flush(); err != nil {
				w.logger.Error("failed to flush WAL", zap.Error(err))
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

	if len(w.worker.data) == 0 {

		return nil
	}
	data := w.worker.data
	w.worker.data = w.worker.data[:0]

	err := w.diskManager.Flush(data)
	for _, rec := range data {
		rec.doneCh <- err
	}

	return err
}

func (w *WAL) triggerFlush() {
	select {
	case w.flushCh <- struct{}{}:
	default:
	}
}
