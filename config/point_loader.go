package config

import (
	"os"
	"sensor-edge/types"

	"gopkg.in/yaml.v3"
)

func LoadPointMappings(file string) ([]types.DevicePointSet, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var sets []types.DevicePointSet
	err = yaml.Unmarshal(data, &sets)
	return sets, err
}
