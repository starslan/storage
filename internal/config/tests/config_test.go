package tests

import (
	"os"
	"reflect"
	"storage/internal/config"
	"testing"

	"github.com/creasty/defaults"
)

func TestLoadConfig_WithDefaultsOnly(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, _ = tmpFile.Write([]byte(`{}`))
	_ = tmpFile.Close()

	cfg, err := config.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Network.Address != "127.0.0.1:3223" {
		t.Errorf("expected default address, got %s", cfg.Network.Address)
	}

	if cfg.Network.MaxConnections != 100 {
		t.Errorf("expected default max connections, got %d", cfg.Network.MaxConnections)
	}
}

func TestLoadConfig_OverrideFromYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	yaml := `
network:
  address: "0.0.0.0:9000"
  max_connections: 200
engine:
  type: "disk"
`
	_, _ = tmpFile.Write([]byte(yaml))
	_ = tmpFile.Close()

	cfg, err := config.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Network.Address != "0.0.0.0:9000" {
		t.Errorf("address not applied")
	}

	if cfg.Network.MaxConnections != 200 {
		t.Errorf("max_connections not applied")
	}

	if cfg.Engine.Type != "disk" {
		t.Errorf("engine type not applied")
	}
}

func TestApplyArguments(t *testing.T) {
	cfg := &config.Config{}
	_ = defaults.Set(cfg)

	// сохраняем оригинальные args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{
		"cmd",
		"-address=1.2.3.4:1234",
		"-max-connections=500",
		"-max_message_size=10KB",
		"-idle_timeout=10m",
	}

	config.ApplyArguments(cfg)

	expected := config.NetworkConfig{
		Address:        "1.2.3.4:1234",
		MaxConnections: 500,
		MaxMessageSize: "10KB",
		IdleTimeout:    "10m",
	}

	if !reflect.DeepEqual(cfg.Network, expected) {
		t.Errorf("unexpected network config: %+v", cfg.Network)
	}
}

func TestLoadConfig_WALOverride(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	yaml := `
wal:
  enable: true
  flushing_batch_size: 50
  flushing_batch_timeout: "200ms"
  max_segment_size: "2MB"
  data_directory: "/tmp/wal"
`
	_, _ = tmpFile.Write([]byte(yaml))
	_ = tmpFile.Close()

	cfg, err := config.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wal := cfg.WALConfig

	if wal.Enable != true {
		t.Errorf("Enable not applied")
	}

	if wal.BatchSize != 50 {
		t.Errorf("BatchSize not applied")
	}

	if wal.BatchTimeout != "200ms" {
		t.Errorf("batchTimeout not applied")
	}

	if wal.MaxSegmentSize != "2MB" {
		t.Errorf("maxSegmentSize not applied")
	}

	if wal.DataDirectory != "/tmp/wal" {
		t.Errorf("DataDirectory not applied")
	}
}
