package config

import (
	"os"
	"sensor-edge/types"

	"gopkg.in/yaml.v3"
)

// LoadPointMappings loads the point mappings from the specified YAML file.
// The default configuration file is located at configs/points.yaml.
func LoadPointMappings(file string) ([]types.DevicePointSet, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var sets []types.DevicePointSet
	err = yaml.Unmarshal(data, &sets)
	return sets, err
}

// 新增：加载三层结构的点位配置
func LoadPointMappingsV2(file string) ([]types.DevicePointSetV2, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var sets []types.DevicePointSetV2
	err = yaml.Unmarshal(data, &sets)
	return sets, err
}
