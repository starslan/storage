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
	//for _, record := range records {
	//
	//}

	return nil
}

func (dm *DiskManager) Load(action func(string) error) error {
	names, err := filepath.Glob(dm.DataPath + "/*.log")
	if err != nil {
		return err
	}

	sort.Strings(names)

	for _, name := range names {
		err = dm.loadFromFile(name, action)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dm *DiskManager) loadFromFile(fileName string, action func(string) error) error {
	file, err := os.Open(fileName)
	defer func() { _ = file.Close() }()
	if err != nil {
		return err
	}

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
	}

	sort.Ints(ids)

	for _, id := range ids {
		err = action(dataMap[id])
		if err != nil {
			return err
		}
	}
	return nil
}
