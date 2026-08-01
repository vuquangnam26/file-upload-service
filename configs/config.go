package configs

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`

	Storage struct {
		Path string `yaml:"path"`
	} `yaml:"storage"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
