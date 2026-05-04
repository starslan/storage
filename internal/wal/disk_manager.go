package wal

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"storage/internal/config"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

type DiskManager struct {
	DataPath       string
	logger         *zap.Logger
	currentFile    *os.File
	lineFileCount  int
	fileNameCount  int
	maxSegmentSize int
}

func NewDiskManager(cfg config.WALConfig, logger *zap.Logger, maxSegmentSize int) *DiskManager {
	return &DiskManager{
		DataPath:       cfg.DataDirectory,
		logger:         logger,
		maxSegmentSize: maxSegmentSize,
	}
}

func (dm *DiskManager) Flush(records []record) error {
	if len(records) == 0 {
		return nil
	}

	writer := bufio.NewWriter(dm.currentFile)

	for i := range records {
		line := records[i].String() + "\n"

		_, err := writer.WriteString(line)
		if err != nil {
			for j := i; j < len(records); j++ {
				if records[j].doneCh != nil {
					records[j].doneCh <- err
				}
			}
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		for i := range records {
			if records[i].doneCh != nil {
				records[i].doneCh <- err
			}
		}
		return err
	}

	if err := dm.currentFile.Sync(); err != nil {
		for i := range records {
			if records[i].doneCh != nil {
				records[i].doneCh <- err
			}
		}
		return err
	}

	for i := range records {
		if records[i].doneCh != nil {
			records[i].doneCh <- nil
		}
	}

	return nil
}

func (dm *DiskManager) Load(action func(string) error) error {
	names, err := filepath.Glob(dm.DataPath + "/*.log")
	if err != nil {
		return err
	}

	sort.Strings(names)

	for i, name := range names {
		last := i == len(names)-1
		err = dm.loadFromFile(name, action, last)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dm *DiskManager) loadFromFile(fileName string, action func(string) error, last bool) error {
	file, err := os.Open(fileName)
	defer func() { _ = file.Close() }()
	if err != nil {
		return err
	}
	lineCount := 0

	scanner := bufio.NewScanner(file)
	ids := make([]int, dm.maxSegmentSize)
	dataMap := map[int]string{}

	for scanner.Scan() {
		line := scanner.Text()
		bf, af, f := strings.Cut(line, "_")
		if !f {
			return errors.New("record not parse")
		}
		id, err := strconv.Atoi(bf)
		if err != nil {
			return err
		}
		ids = append(ids, id)
		dataMap[id] = af
		if last {
			lineCount++
		}
	}

	sort.Ints(ids)

	for _, id := range ids {
		err = action(dataMap[id])
		if err != nil {
			return err
		}
	}

	if last {
		dm.lineFileCount = lineCount
		dm.currentFile = file
	}
	return nil
}
