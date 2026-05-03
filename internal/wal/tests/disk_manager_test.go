package tests

import (
	"os"
	"path/filepath"
	"storage/internal/wal"
	"testing"
)

const WorkerPoolSize = 3

func TestLoad_Success(t *testing.T) {
	dir := t.TempDir()

	files := []string{"0000000001.log", "0000000002.log", "0000000003 .log"}
	for _, f := range files {
		_ = os.WriteFile(filepath.Join(dir, f), []byte("test"), 0644)
	}

	dm := &wal.DiskManager{
		DataPath: dir,
	}

	err := dm.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
