package config

import (
	"os"
	"sync"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"
)

var (
	instance *Config
	once     sync.Once
)

type Config struct {
	Network NetworkConfig `yaml:"network"`
	Engine  EngineConfig  `yaml:"engine"`
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

func LoadConfig(configPath string) (*Config, error) {
	var err error
	once.Do(func() {
		var cfg Config
		if defaultsErr := defaults.Set(&cfg); defaultsErr != nil {
			err = defaultsErr
			return
		}
		data, readErr := os.ReadFile(configPath)
		if readErr != nil {
			err = readErr
			return
		}

		if unmarshalErr := yaml.Unmarshal(data, &cfg); unmarshalErr != nil {
			err = unmarshalErr
			return
		}

		instance = &cfg
	})

	return instance, err
}
