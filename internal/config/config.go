package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"time"
)

const configPath = "config.yaml"

type Config struct {
	Logger     Logger     `yaml:"Logger"`
	Postgres   Postgres   `yaml:"Postgres"`
	HTTPServer HTTPServer `yaml:"HTTPServer"`
	Workers    Workers    `yaml:"Workers"`
}

type Logger struct {
	Level uint8 `yaml:"Level"`
}

type Postgres struct {
	ConnString string `yaml:"ConnString"`
}

type HTTPServer struct {
	Address string `yaml:"Address"`
}

type Workers struct {
	OutboxRelyWorker OutboxRely `yaml:"OutboxRely"`
}

type OutboxRely struct {
	KafkaBrokers []string      `yaml:"KafkaBrokers"`
	EventsLimit  int64         `yaml:"EventsLimit"`
	Interval     time.Duration `yaml:"Interval"`
}

func Read() (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("error open config file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("error closing config file: %v", err)
		}
	}()

	var config Config
	if err := yaml.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("error decoding config file: %v", err)
	}

	return &config, nil
}
