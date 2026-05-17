package wal

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"storage/internal/config"
)

type mockDiskManager struct {
	loadID    *int
	loadErr   error
	flushErr  error
	mu        sync.Mutex
	flushed   [][]Record
	flushedCh chan []Record
}

func (m *mockDiskManager) Load(func(string) error) (*int, error) {
	return m.loadID, m.loadErr
}

func (m *mockDiskManager) Flush(records []Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cp := make([]Record, len(records))
	copy(cp, records)
	m.flushed = append(m.flushed, cp)
	if m.flushedCh != nil {
		select {
		case m.flushedCh <- cp:
		default:
		}
	}
	return m.flushErr
}

func TestWriteFlushesOnBatchSize(t *testing.T) {
	cfg := config.WALConfig{
		DataDirectory:  t.TempDir() + "/",
		BatchTimeout:   "1s",
		BatchSize:      2,
		MaxSegmentSize: "1MB",
	}
	w, err := NewWAL(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewWAL returned error: %v", err)
	}
	mock := &mockDiskManager{flushedCh: make(chan []Record, 10)}
	w.diskManager = mock

	if err := w.Start(func(string) error { return nil }); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer w.Stop()

	if err := w.Write(context.Background(), "first"); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if err := w.Write(context.Background(), "second"); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	var flushed []Record
	timeout := time.After(2 * time.Second)
	for len(flushed) < 2 {
		select {
		case batch := <-mock.flushedCh:
			flushed = append(flushed, batch...)
		case <-timeout:
			t.Fatalf("expected 2 flushed records, got %d", len(flushed))
		}
	}

	if flushed[0].id != 1 || flushed[1].id != 2 {
		t.Fatalf("unexpected record IDs: got %d and %d, want 1 and 2", flushed[0].id, flushed[1].id)
	}
	if flushed[0].data != "first" || flushed[1].data != "second" {
		t.Fatalf("unexpected flushed data: %+v", flushed)
	}
}

func TestStartSetsInitialIDFromLoad(t *testing.T) {
	cfg := config.WALConfig{
		DataDirectory:  t.TempDir() + "/",
		BatchTimeout:   "1s",
		BatchSize:      4,
		MaxSegmentSize: "1MB",
	}
	w, err := NewWAL(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewWAL returned error: %v", err)
	}
	loadID := 5
	mock := &mockDiskManager{loadID: &loadID, flushedCh: make(chan []Record, 10)}
	w.diskManager = mock

	if err := w.Start(func(string) error { return nil }); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer w.Stop()

	if err := w.Write(context.Background(), "next"); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	w.triggerFlush()

	var flushed []Record
	select {
	case flushed = <-mock.flushedCh:
	case <-time.After(2 * time.Second):
		t.Fatal("flush did not occur within timeout")
	}

	if len(flushed) != 1 {
		t.Fatalf("expected 1 flushed record, got %d", len(flushed))
	}
	if flushed[0].id != 6 {
		t.Fatalf("unexpected record ID: got %d, want 6", flushed[0].id)
	}
	if flushed[0].data != "next" {
		t.Fatalf("unexpected flushed data: %+v", flushed)
	}
}
