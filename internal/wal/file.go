package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var logFileRegexp = regexp.MustCompile(`^*\d{10}\.log$`)

func lastLogFile(dir string) (*os.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var (
		maxVal  int64 = -1
		maxFile string
	)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()

		v, err := strconv.ParseInt(name, 10, 64)
		if err != nil {
			continue
		}

		if v > maxVal {
			maxVal = v
			maxFile = name
		}
	}

	if maxFile == "" {
		return startLogFile(dir)
	}

	fullPath := filepath.Join(dir, maxFile)

	f, err := os.OpenFile(fullPath, os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func startLogFile(dir string) (*os.File, error) {
	name := fmt.Sprintf("%010d", 1)
	fullPath := filepath.Join(dir, name)

	f, err := os.Create(fullPath)
	if err != nil {
		return nil, err
	}

	return f, nil
}
