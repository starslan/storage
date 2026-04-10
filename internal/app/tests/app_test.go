package tests

import (
	"storage/internal/app"
	"storage/internal/config"
	"testing"

	"go.uber.org/zap"
)

func TestNewApp_Success(t *testing.T) {
	cfg := &config.Config{
		Network: config.NetworkConfig{
			Address:        "127.0.0.1:3223",
			MaxConnections: 10,
			MaxMessageSize: "4KB",
			IdleTimeout:    "5m",
		},
		Engine: config.EngineConfig{
			Type: "in_memory",
		},
		Parser: config.ParserConfig{
			MaxQueryLength: 200,
		},
		Logger: config.LoggerConfig{
			Level:  "debug",
			Output: "stdout",
		},
	}

	logger := zap.NewNop()

	appInstance, err := app.NewApp(cfg, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if appInstance == nil {
		t.Fatal("expected app instance, got nil")
	}

	if appInstance.DB == nil {
		t.Error("DB should be initialized")
	}

	if appInstance.Logger == nil {
		t.Error("Logger should be initialized")
	}
}
