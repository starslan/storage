package wal

import (
	"os"
	"storage/internal/config"
	"sync/atomic"

	"go.uber.org/zap"
)

type Buffer struct {
	signals     chan struct{}
	data        chan []byte
	idx         atomic.Int64
	currentFile *os.File
	logger      *zap.Logger
}

func NewBuffer(cfg config.WALConfig, logger *zap.Logger) (*Buffer, error) {
	return &Buffer{}, nil
}
