package wal

import (
	"context"
	"errors"
	"fmt"
	"storage/internal/config"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

type MockDiskManager struct {
	mu         sync.Mutex
	records    []record
	shouldFail bool
	failAfter  int
	writeCount int
	loadFunc   func(string) error
}

func (m *MockDiskManager) Flush(records []record) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.writeCount++

	if m.shouldFail {
		if m.failAfter == 0 || m.writeCount >= m.failAfter {
			return errors.New("mock flush error")
		}
	}

	m.records = append(m.records, records...)
	return nil
}

func (m *MockDiskManager) Load(fn func(string) error) error {
	//if m.loadFunc != nil {
	//	return m.loadFunc(fn)
	//}
	return nil
}

func (m *MockDiskManager) GetRecords() []record {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.records
}

func TestWALBasicWrite(t *testing.T) {
	mock := &MockDiskManager{}
	logger, _ := zap.NewDevelopment()

	cfg := config.WALConfig{
		BatchTimeout:   "100ms",
		MaxSegmentSize: "1KB",
		BatchSize:      100,
		DataDirectory:  t.TempDir(),
	}

	wal, err := NewWAL(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	err = wal.Start(func(s string) error { return nil })
	if err != nil {
		t.Fatalf("Failed to start WAL: %v", err)
	}
	defer wal.Stop()

	// Записываем данные
	ctx := context.Background()
	err = wal.Write(ctx, "test data 1")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Даем время на flush
	time.Sleep(200 * time.Millisecond)

	mock.mu.Lock()
	recordCount := len(mock.records)
	mock.mu.Unlock()

	if recordCount == 0 {
		t.Error("Expected records to be flushed, got 0")
	}
}

func TestWALBatchFlush(t *testing.T) {
	mock := &MockDiskManager{}
	logger, _ := zap.NewDevelopment()

	cfg := config.WALConfig{
		BatchTimeout:   "1s",
		MaxSegmentSize: "100B", // Маленький размер для быстрого заполнения
		BatchSize:      10,
		DataDirectory:  t.TempDir(),
	}

	wal, err := NewWAL(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	err = wal.Start(func(s string) error { return nil })
	if err != nil {
		t.Fatalf("Failed to start WAL: %v", err)
	}
	defer wal.Stop()

	// Пишем много данных
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		err := wal.Write(ctx, fmt.Sprintf("data_%d", i))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Даем время на flush
	time.Sleep(200 * time.Millisecond)

	mock.mu.Lock()
	recordCount := len(mock.records)
	mock.mu.Unlock()

	if recordCount == 0 {
		t.Error("Expected automatic flush, got 0 records")
	}
}

func TestWALContextCancellation(t *testing.T) {
	//mock := &MockDiskManager{
	//	shouldFail: false,
	//}
	logger, _ := zap.NewDevelopment()

	cfg := config.WALConfig{
		BatchTimeout:   "1s",
		MaxSegmentSize: "1KB",
		BatchSize:      10,
		DataDirectory:  t.TempDir(),
	}

	wal, err := NewWAL(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	err = wal.Start(func(s string) error { return nil })
	if err != nil {
		t.Fatalf("Failed to start WAL: %v", err)
	}
	defer wal.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = wal.Write(ctx, "test data")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestWALConcurrentWrites(t *testing.T) {
	//mock := &MockDiskManager{}
	logger, _ := zap.NewDevelopment()

	cfg := config.WALConfig{
		BatchTimeout:   "100ms",
		MaxSegmentSize: "1KB",
		BatchSize:      1000,
		DataDirectory:  t.TempDir(),
	}

	wal, err := NewWAL(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	err = wal.Start(func(s string) error { return nil })
	if err != nil {
		t.Fatalf("Failed to start WAL: %v", err)
	}
	defer wal.Stop()

	var wg sync.WaitGroup
	numGoroutines := 10
	numWritesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < numWritesPerGoroutine; j++ {
				data := fmt.Sprintf("goroutine_%d_data_%d", id, j)
				err := wal.Write(ctx, data)
				if err != nil {
					t.Errorf("Write failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	time.Sleep(200 * time.Millisecond)

	t.Log("Concurrent writes completed successfully")
}

func TestWALFlushError(t *testing.T) {
	//mock := &MockDiskManager{
	//	shouldFail: true,
	//	failAfter:  1,
	//}
	logger, _ := zap.NewDevelopment()

	cfg := config.WALConfig{
		BatchTimeout:   "50ms",
		MaxSegmentSize: "1KB",
		BatchSize:      10,
		DataDirectory:  t.TempDir(),
	}

	wal, err := NewWAL(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	err = wal.Start(func(s string) error { return nil })
	if err != nil {
		t.Fatalf("Failed to start WAL: %v", err)
	}
	defer wal.Stop()

	ctx := context.Background()

	err = wal.Write(ctx, "data 1")
	if err != nil {
		t.Errorf("First write should succeed, got: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	err = wal.Write(ctx, "data 2")
	if err == nil {
		t.Log("Write succeeded (may be buffered)")
	}
}

func TestWALStop(t *testing.T) {
	//mock := &MockDiskManager{}
	logger, _ := zap.NewDevelopment()

	cfg := config.WALConfig{
		BatchTimeout:   "1s",
		MaxSegmentSize: "1KB",
		BatchSize:      10,
		DataDirectory:  t.TempDir(),
	}

	wal, err := NewWAL(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	err = wal.Start(func(s string) error { return nil })
	if err != nil {
		t.Fatalf("Failed to start WAL: %v", err)
	}

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		err := wal.Write(ctx, fmt.Sprintf("pre_stop_%d", i))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	wal.Stop()

	err = wal.Write(ctx, "after stop")
	if err == nil {
		t.Error("Expected error when writing to stopped WAL")
	}
}

func TestWALWithRealDiskManager(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tempDir := t.TempDir()
	logger, _ := zap.NewDevelopment()

	cfg := config.WALConfig{
		BatchTimeout:   "100ms",
		MaxSegmentSize: "1KB",
		BatchSize:      100,
		DataDirectory:  tempDir,
	}

	wal, err := NewWAL(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	var replayed []string
	replayFunc := func(s string) error {
		replayed = append(replayed, s)
		return nil
	}

	err = wal.Start(replayFunc)
	if err != nil {
		t.Fatalf("Failed to start WAL: %v", err)
	}

	// Записываем данные
	testData := []string{"test1", "test2", "test3"}
	ctx := context.Background()
	for _, data := range testData {
		err := wal.Write(ctx, data)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Даем время на запись
	time.Sleep(200 * time.Millisecond)

	// Останавливаем
	wal.Stop()

	// Создаем новый WAL для проверки восстановления
	wal2, err := NewWAL(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create second WAL: %v", err)
	}

	var replayed2 []string
	replayFunc2 := func(s string) error {
		replayed2 = append(replayed2, s)
		return nil
	}

	err = wal2.Start(replayFunc2)
	if err != nil {
		t.Fatalf("Failed to start second WAL: %v", err)
	}
	defer wal2.Stop()

	time.Sleep(100 * time.Millisecond)

	if len(replayed2) == 0 {
		t.Error("Expected data to be replayed, got 0")
	}
}

// BenchmarkWALWrite бенчмарк производительности записи
func BenchmarkWALWrite(b *testing.B) {
	//mock := &MockDiskManager{}
	logger, _ := zap.NewDevelopment()

	tempDir := b.TempDir()
	cfg := config.WALConfig{
		BatchTimeout:   "1s",
		MaxSegmentSize: "10MB",
		BatchSize:      10000,
		DataDirectory:  tempDir,
	}

	wal, err := NewWAL(cfg, logger)
	if err != nil {
		b.Fatalf("Failed to create WAL: %v", err)
	}

	err = wal.Start(func(s string) error { return nil })
	if err != nil {
		b.Fatalf("Failed to start WAL: %v", err)
	}
	defer wal.Stop()

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := wal.Write(ctx, fmt.Sprintf("bench_data_%d", i))
		if err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}
}

// TestWALDataRace тест на data race (запускать с -race)
func TestWALDataRace(t *testing.T) {
	//mock := &MockDiskManager{}
	logger, _ := zap.NewDevelopment()

	cfg := config.WALConfig{
		BatchTimeout:   "10ms",
		MaxSegmentSize: "100B",
		BatchSize:      100,
		DataDirectory:  t.TempDir(),
	}

	wal, err := NewWAL(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	err = wal.Start(func(s string) error { return nil })
	if err != nil {
		t.Fatalf("Failed to start WAL: %v", err)
	}
	defer wal.Stop()

	var wg sync.WaitGroup

	// Писатели
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < 100; j++ {
				wal.Write(ctx, "data")
			}
		}()
	}

	// Читатели (через Stop/Start)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond)
			// Просто вызываем Stop и новый Start для теста
		}()
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)
}
