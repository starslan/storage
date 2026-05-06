package wal

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"storage/internal/config"
	"storage/pkg/utils"
	"strconv"
	"strings"
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

func (dm *DiskManager) Flush(records []record) error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	if len(records) == 0 {
		return nil
	}

	writer := bufio.NewWriter(dm.currentFile)

	for i := range records {
		line := records[i].String()

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

	info, err := dm.currentFile.Stat()
	if err != nil {
		return err
	}

	if info.Size() >= dm.maxSegmentSize {
		return dm.rotateLogFile()
	}

	return nil
}

func (dm *DiskManager) Load(action func(string) error) (*int, error) {
	names, err := filepath.Glob(dm.DataPath + "/*.log")
	if err != nil {
		return nil, err
	}

	if len(names) == 0 {
		return nil, dm.rotateLogFile()
	}

	sort.Strings(names)

	var lastID *int
	for i, name := range names {
		last := i == len(names)-1
		var id *int
		if id, err = dm.loadFromFile(name, action, last); err != nil {
			return nil, err
		}
		if last && id != nil {
			lastID = id
		}
	}

	return lastID, nil

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

func (dm *DiskManager) loadFromFile(fileName string, action func(string) error, last bool) (*int, error) {
	file, err := os.Open(fileName)
	defer func() {
		if !last {
			_ = file.Close()
		}
	}()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	ids := make([]int, 0, dm.batchSize)
	dataMap := map[int]string{}
	var lastId int

	for scanner.Scan() {
		line := scanner.Text()
		bf, af, f := strings.Cut(line, "_")
		if !f {
			return nil, errors.New("record not parse")
		}
		id, err := strconv.Atoi(bf)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
		dataMap[id] = af
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if last {
		_ = file.Close()

		file, err = os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}

		dm.currentFile = file
		numStr := strings.TrimSuffix(filepath.Base(file.Name()), ".log")
		number, err := strconv.Atoi(numStr)
		if err != nil {
			return nil, err
		}
		dm.fileNameCount = number
	}

	if len(ids) == 0 {
		return nil, nil
	}

	sort.Ints(ids)

	if last {
		lastId = ids[len(ids)-1]
	}

	for _, id := range ids {
		if err := action(dataMap[id]); err != nil {
			return nil, err
		}
	}
	return &lastId, nil
}

func (dm *DiskManager) setCurrentFile(file *os.File) error {
	dm.currentFile = file
	numStr := strings.TrimSuffix(filepath.Base(file.Name()), ".log")

	number, err := strconv.Atoi(numStr)
	if err != nil {
		return err
	}
	dm.fileNameCount = number
	return nil
}
