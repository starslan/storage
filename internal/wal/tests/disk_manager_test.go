package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"go.uber.org/zap"

	"storage/internal/config"
	"storage/internal/wal"
)

func TestLoadCreatesNewSegmentWhenEmpty(t *testing.T) {
	dir := t.TempDir()

	cfg := config.WALConfig{
		DataDirectory:  filepath.Clean(dir) + string(os.PathSeparator),
		BatchSize:      4,
		MaxSegmentSize: "1MB",
	}

	dm := wal.NewDiskManager(cfg, zap.NewNop())
	t.Cleanup(func() {
		if closer, ok := any(dm).(interface{ Close() error }); ok {
			_ = closer.Close()
		} else {
			dm = nil
			runtime.GC()
		}
	})

	lastID, err := dm.Load(func(string) error { return nil })
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if lastID != nil {
		t.Fatalf("expected lastID to be nil, got %v", *lastID)
	}

	files, err := filepath.Glob(filepath.Join(cfg.DataDirectory, "*.log"))
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one log file to be created, got %d", len(files))
	}
	if filepath.Base(files[0]) != "00000000001.log" {
		t.Fatalf("unexpected log file name: %s", filepath.Base(files[0]))
	}
}

func TestLoadReadsExistingSegments(t *testing.T) {
	dir := t.TempDir()

	// Prepare two WAL segments with unsorted ids to verify sorting.
	file1 := filepath.Join(dir, fmt.Sprintf("%011d.log", 1))
	file2 := filepath.Join(dir, fmt.Sprintf("%011d.log", 2))

	if err := os.WriteFile(file1, []byte("2_a\n1_b\n"), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", file1, err)
	}
	if err := os.WriteFile(file2, []byte("5_e\n4_d\n"), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", file2, err)
	}

	cfg := config.WALConfig{
		DataDirectory:  filepath.Clean(dir) + string(os.PathSeparator),
		BatchSize:      4,
		MaxSegmentSize: "1MB",
	}

	dm := wal.NewDiskManager(cfg, zap.NewNop())
	t.Cleanup(func() {
		if closer, ok := any(dm).(interface{ Close() error }); ok {
			_ = closer.Close()
		} else {
			dm = nil
			runtime.GC()
		}
	})

	var records []string
	lastID, err := dm.Load(func(val string) error {
		records = append(records, val)
		return nil
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	expected := []string{"b", "a", "d", "e"}
	if !reflect.DeepEqual(records, expected) {
		t.Fatalf("unexpected records order: got %v, want %v", records, expected)
	}

	if lastID == nil || *lastID != 5 {
		t.Fatalf("unexpected lastID: got %v, want %d", lastID, 5)
	}
}
