package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

const configPath = "config.yaml"

type Config struct {
	Logger     Logger     `yaml:"Logger"`
	Postgres   Postgres   `yaml:"Postgres"`
	HTTPServer HTTPServer `yaml:"HTTPServer"`
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
