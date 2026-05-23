package config

import (
	"flag"
	"os"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Network   NetworkConfig `yaml:"network"`
	Engine    EngineConfig  `yaml:"engine"`
	Parser    ParserConfig  `yaml:"parse"`
	Logger    LoggerConfig  `yaml:"logging"`
	WALConfig WALConfig     `yaml:"wal"`
}

type NetworkConfig struct {
	Address        string `yaml:"address" default:"127.0.0.1:3223"`
	MaxConnections int    `yaml:"max_connections" default:"100"`
	MaxMessageSize string `yaml:"max_message_size" default:"4KB"`
	IdleTimeout    string `yaml:"idle_timeout" default:"5m"`
}

type EngineConfig struct {
	Type string `yaml:"type" default:"in_memory"`
}

type ParserConfig struct {
	MaxQueryLength int `yaml:"max_query_length" default:"200"`
}

type LoggerConfig struct {
	Level  string `yaml:"level" default:"info"`
	Output string `yaml:"output" default:"stdout"`
}

func LoadConfig(configPath string) (*Config, error) {
	var cfg Config

	if err := defaults.Set(&cfg); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func ApplyArguments(cfg *Config) {
	address := flag.String("address", "", "network address")
	maxConn := flag.Int("max-connections", 0, "max connections")
	maxMessageSize := flag.String("max_message_size", "", "max message size")
	idleTimeout := flag.String("idle_timeout", "", "idle timeout TCP")

	flag.Parse()

	if *address != "" {
		cfg.Network.Address = *address
	}
	if *maxConn != 0 {
		cfg.Network.MaxConnections = *maxConn
	}
	if *maxMessageSize != "" {
		cfg.Network.MaxMessageSize = *maxMessageSize
	}

	if *idleTimeout != "" {
		cfg.Network.IdleTimeout = *idleTimeout
	}
}

type WALConfig struct {
	Enable         bool   `yaml:"enable" default:"false"`
	BatchSize      int    `yaml:"flushing_batch_size" default:"20"`
	BatchTimeout   string `yaml:"flushing_batch_timeout" default:"100ms"`
	MaxSegmentSize string `yaml:"max_segment_size" default:"1MB"`
	DataDirectory  string `yaml:"data_directory" default:"./data/wal"`
}
