package wal

import "storage/internal/config"

type DiskManager struct {
	DataPath string
}

func NewDiskManager(cfg config.WALConfig) *DiskManager {
	return &DiskManager{
		DataPath: cfg.DataDirectory,
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
