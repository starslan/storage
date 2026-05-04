package tests

import (
	"os"
	"path/filepath"
	"storage/internal/wal"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	dir := t.TempDir()

	files := []string{
		"0000000001.log",
		"0000000002.log",
	}

	content := []string{
		"2_SET key2 value2\n1_SET key1 value1\n3_SET key3 value3",
		"4_SET key4 value4",
	}

	for i, f := range files {
		err := os.WriteFile(filepath.Join(dir, f), []byte(content[i]), 0644)
		if err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	var loaded []string

	dm := &wal.DiskManager{
		DataPath: dir,
	}

	err := dm.Load(func(s string) error {
		loaded = append(loaded, s)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"SET key1 value1",
		"SET key2 value2",
		"SET key3 value3",
		"SET key4 value4",
	}

	if len(loaded) != len(expected) {
		t.Fatalf("expected %d records, got %d", len(expected), len(loaded))
	}

	for i := range expected {
		if loaded[i] != expected[i] {
			t.Fatalf("unexpected record at %d: got %q, want %q", i, loaded[i], expected[i])
		}
	}
}
