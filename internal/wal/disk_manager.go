package wal

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"storage/internal/config"
	"storage/pkg/utils"
	"sync"

	"go.uber.org/zap"
)

var logFileRegexp = regexp.MustCompile(`^*\d{10}\.log$`)

type DiskManager struct {
	DataPath       string
	logger         *zap.Logger
	currentFile    *os.File
	fileNameCount  int
	batchSize      int
	maxSegmentSize int64
	mutex          sync.Mutex
}

func NewDiskManager(cfg config.WALConfig, logger *zap.Logger) *DiskManager {
	return &DiskManager{
		DataPath:       cfg.DataDirectory,
		logger:         logger,
		batchSize:      cfg.BatchSize,
		maxSegmentSize: int64(utils.ParseSize(cfg.MaxSegmentSize)),
	}
}

func (dm *DiskManager) Flush(records []Record) error {
	if len(records) == 0 {
		return nil
	}

	writer := bufio.NewWriter(dm.currentFile)

	for i := range records {
		line := records[i].String()

		_, err := writer.WriteString(line)
		if err != nil {
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	if err := dm.currentFile.Sync(); err != nil {
		return err
	}

	info, err := dm.currentFile.Stat()
	if err != nil {
		return err
	}

	if info.Size() >= dm.maxSegmentSize {
		return dm.rotateLogFile()
	}

	return nil
}

func (dm *DiskManager) rotateLogFile() error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	if dm.currentFile != nil {
		if err := dm.currentFile.Close(); err != nil {
			return err
		}
	}
	dm.fileNameCount++
	file, err := os.Create(dm.DataPath + fmt.Sprintf("%011d.log", dm.fileNameCount))
	if err != nil {
		return err
	}
	dm.currentFile = file
	return nil
}
