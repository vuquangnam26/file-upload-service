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

	MinIO struct {
		Endpoint  string `yaml:"endpoint"`
		AccessKey string `yaml:"access_key"`
		SecretKey string `yaml:"secret_key"`
		UseSSL    bool   `yaml:"use_ssl"`
		Bucket    string `yaml:"bucket"`
	} `yaml:"minio"`

	Kafka struct {
		Brokers []string `yaml:"brokers"`
		GroupID string   `yaml:"group_id"`
		Topics  struct {
			FileUploaded  string `yaml:"file_uploaded"`
			FileProcessed string `yaml:"file_processed"`
		} `yaml:"topics"`
	} `yaml:"kafka"`
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
