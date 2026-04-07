package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"storage/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(cfg config.LoggerConfig) (*zap.Logger, error) {
	level := zapcore.InfoLevel

	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		fmt.Println(err)
	}

	outputPaths := []string{"stdout"}
	if cfg.Output != "" && cfg.Output != "stdout" && cfg.Output != "stderr" {
		// Создаем директорию для файла лога
		dir := filepath.Dir(cfg.Output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		outputPaths = append([]string{cfg.Output}, outputPaths...)
	}
	zapCfg := zap.Config{
		OutputPaths: []string{cfg.Output, "stdout"},
		Level:       zap.NewAtomicLevelAt(level),
		Encoding:    "json",
	}
	logger, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	return logger, nil
}
