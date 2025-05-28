package config

import (
	"os"

	"sensor-edge/types"

	"gopkg.in/yaml.v3"
)

func LoadConfig(filePath string) (*types.Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cfg types.Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
