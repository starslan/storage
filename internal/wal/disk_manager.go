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

	"go.uber.org/zap"
)

var logFileRegexp = regexp.MustCompile(`^*\d{10}\.log$`)

type DiskManager struct {
	DataPath       string
	logger         *zap.Logger
	currentFile    *os.File
	lineFileCount  int
	fileNameCount  int
	batchSize      int
	maxSegmentSize int64
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

	fmt.Println(info.Size(), dm.maxSegmentSize)
	if info.Size() >= dm.maxSegmentSize {
		return dm.rotateLogFile()
	}

	return nil
}

func (dm *DiskManager) Load(action func(string) error) error {
	names, err := filepath.Glob(dm.DataPath + "/*.log")
	if err != nil {
		return err
	}

	if len(names) == 0 {
		return dm.rotateLogFile()
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

func (dm *DiskManager) rotateLogFile() error {
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
	dm.lineFileCount = 0
	return nil
}

func (dm *DiskManager) loadFromFile(fileName string, action func(string) error, last bool) error {
	file, err := os.Open(fileName)
	defer func() { _ = file.Close() }()
	if err != nil {
		return err
	}
	dm.lineFileCount = 0

	scanner := bufio.NewScanner(file)
	ids := make([]int, 0, dm.batchSize)
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
			dm.lineFileCount++
		}
	}

	if last {
		err = dm.parseLastFile(file)
		if err != nil {
			return err
		}
	}

	if len(ids) == 0 {
		return nil
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

func (dm *DiskManager) parseLastFile(file *os.File) error {
	dm.currentFile = file
	numStr := strings.TrimSuffix(filepath.Base(file.Name()), ".log")

	number, err := strconv.Atoi(numStr)
	if err != nil {
		return err
	}
	dm.fileNameCount = number
	return nil
}
